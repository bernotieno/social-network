import React, { useState, useEffect, useRef } from 'react';
import { groupAPI } from '@/utils/api';
import { useAuth } from '@/hooks/useAuth';
import { initializeSocket, getSocket, subscribeToMessages, joinChatRoom, leaveChatRoom } from '@/utils/socket';
import { getUserProfilePictureUrl, getFallbackAvatar } from '@/utils/images';
import { useAlert } from '@/contexts/AlertContext';
import styles from '@/styles/GroupChat.module.css';
import emojis from "@/components/emojis";
import stylesB from '@/styles/Chat.module.css'

export default function GroupChat({ groupId, isVisible }) {
  const { showError } = useAlert();
  const [messages, setMessages] = useState([]);
  const [newMessage, setNewMessage] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [isSending, setIsSending] = useState(false);
  const { user } = useAuth();
  const messagesEndRef = useRef(null);
  const chatContainerRef = useRef(null);

  // Scroll to bottom of messages
  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  // Fetch group messages
  const fetchMessages = async () => {
    if (!groupId) return;

    setIsLoading(true);
    try {
      const response = await groupAPI.getGroupMessages(groupId);
      setMessages(response.data.data.messages || []);
    } catch (error) {
      console.error('Error fetching group messages:', error);
    } finally {
      setIsLoading(false);
    }
  };

  // Send message
  const handleSendMessage = async (e) => {
    e.preventDefault();
    if (!newMessage.trim() || isSending) return;

    const messageContent = newMessage.trim();
    setIsSending(true);

    // Create optimistic message object
    const optimisticMessage = {
      id: `temp-${Date.now()}`, // Temporary ID
      content: messageContent,
      senderId: user?.id,
      groupId: groupId,
      createdAt: new Date().toISOString(),
      sender: {
        id: user?.id,
        fullName: user?.fullName,
        username: user?.username,
        profilePicture: user?.profilePicture
      }
    };

    // Add message to local state immediately for instant feedback
    setMessages(prev => [optimisticMessage, ...prev]);
    setNewMessage('');

    try {
      // Send to backend
      const response = await groupAPI.sendGroupMessage(groupId, messageContent);

      // Replace optimistic message with real message from server
      if (response.data && response.data.data && response.data.data.message) {
        const realMessage = response.data.data.message;
        setMessages(prev => prev.map(msg =>
          msg.id === optimisticMessage.id ? realMessage : msg
        ));
      }
    } catch (error) {
      console.error('Error sending message:', error);
      // Remove optimistic message on error
      setMessages(prev => prev.filter(msg => msg.id !== optimisticMessage.id));
      // Restore message content
      setNewMessage(messageContent);
      showError('Failed to send message. Please try again.');
    } finally {
      setIsSending(false);
    }
  };

  // Handle emoji insertion
  const insertEmoji = (emoji) => {
    setNewMessage(prev => prev + emoji);
  };

  // Initialize WebSocket connection
  useEffect(() => {
    const socket = initializeSocket();
    if (!socket) {
      console.error('Failed to initialize WebSocket for group chat');
    }
  }, []);

  // WebSocket message handler
  useEffect(() => {
    if (!groupId) return;

    const roomId = `group-${groupId}`;

    // Join the chat room
    joinChatRoom(roomId);

    // Subscribe to messages
    const unsubscribe = subscribeToMessages((data) => {
      if (data.roomId === roomId) {
        const messageData = data.message;
        const newMsg = {
          id: messageData.id,
          content: messageData.content,
          senderId: messageData.sender,
          groupId: messageData.groupId,
          createdAt: messageData.timestamp,
          sender: messageData.senderInfo
        };

        // Only add if it's not already in the list (avoid duplicates from optimistic updates)
        setMessages(prev => {
          const exists = prev.some(msg => msg.id === newMsg.id);
          if (!exists) {
            return [newMsg, ...prev];
          }
          return prev;
        });
      }
    });

    return () => {
      // Leave room and unsubscribe
      leaveChatRoom(roomId);
      unsubscribe();
    };
  }, [groupId]);

  // Fetch messages when component mounts or groupId changes
  useEffect(() => {
    if (isVisible && groupId) {
      fetchMessages();
    }
  }, [groupId, isVisible]);

  // Scroll to bottom when messages change
  useEffect(() => {
    scrollToBottom();
  }, [messages]);


  const emojiList = emojis();
  const [showEmojis, setShowEmojis] = useState(false);


  if (!isVisible) return null;


  return (
    <div className={styles.groupChat}>
      <div className={styles.chatHeader}>
        <h3>Group Chat</h3>
      </div>

      <div className={styles.messagesContainer} ref={chatContainerRef}>
        {isLoading ? (
          <div className={styles.loading}>Loading messages...</div>
        ) : messages.length === 0 ? (
          <div className={styles.noMessages}>
            No messages yet. Start the conversation!
          </div>
        ) : (
          <div className={styles.messagesList}>
            {messages.slice().reverse().map((message) => (
              <div
                key={message.id}
                className={`${styles.message} ${
                  message.senderId === user?.id ? styles.ownMessage : styles.otherMessage
                }`}
              >
                <div className={styles.messageHeader}>
                  <img
                    src={message.sender?.profilePicture ? getUserProfilePictureUrl(message.sender) : getFallbackAvatar(message.sender)}
                    alt={message.sender?.fullName}
                    className={styles.senderAvatar}
                    onError={(e) => {
                      e.target.src = getFallbackAvatar(message.sender);
                    }}
                  />
                  <span className={styles.senderName}>
                    {message.senderId === user?.id ? 'You' : message.sender?.fullName}
                  </span>
                  <span className={styles.messageTime}>
                    {new Date(message.createdAt).toLocaleTimeString([], {
                      hour: '2-digit',
                      minute: '2-digit'
                    })}
                  </span>
                </div>
                <div className={styles.messageContent}>
                  {message.content}
                </div>
              </div>
            ))}
            <div ref={messagesEndRef} />
          </div>
        )}
      </div>

      <form onSubmit={handleSendMessage} className={styles.messageForm}>
        {showEmojis && (
            <div className={stylesB.emojiBarGroup}>
              {emojiList.map((emoji,key) => (
                  <button
                      key={key}
                      type="button"
                      className={stylesB.emojiButton}
                      onClick={() => insertEmoji(emoji)}
                  >
                    {emoji}
                  </button>
              ))}
            </div>
        )}

        <div className={styles.inputContainer}>
          <input
            type="text"
            value={newMessage}
            onChange={(e) => setNewMessage(e.target.value)}
            placeholder="Type a message..."
            className={styles.messageInput}
            disabled={isSending}
          />

          {/*toggle emoji box*/}
          <button
              type="button"
              onClick={() => setShowEmojis(!showEmojis)}
              className={stylesB.emojiToggleButton}
          >
            {showEmojis ? 'hide emojis' : 'show more emojis'}
          </button>

          <button
            type="submit"
            disabled={!newMessage.trim() || isSending}
            className={styles.sendButton}
          >
            {isSending ? 'Sending...' : 'Send'}
          </button>
        </div>
      </form>
    </div>
  );
}
