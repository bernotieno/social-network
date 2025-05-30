'use client';

import { useState, useEffect, useRef } from 'react';
import Image from 'next/image';
import { useAuth } from '@/hooks/useAuth';
import { userAPI, messageAPI } from '@/utils/api';
import {
  initializeSocket,
  getSocket,
  joinChatRoom,
  leaveChatRoom,
  sendMessage,
  subscribeToMessages
} from '@/utils/socket';
import Button from '@/components/Button';
import ProtectedRoute from '@/components/ProtectedRoute';
import styles from '@/styles/Chat.module.css';

export default function Chat() {
  const { user } = useAuth();
  const [contacts, setContacts] = useState([]);
  const [selectedContact, setSelectedContact] = useState(null);
  const [messages, setMessages] = useState([]);
  const [newMessage, setNewMessage] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [isConnected, setIsConnected] = useState(false);

  const messagesEndRef = useRef(null);

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  // Initialize socket connection
  useEffect(() => {
    console.log('Initializing socket connection...');
    const socket = initializeSocket();
    if (socket) {
      console.log('Socket initialized successfully');

      // Add connection event listeners
      socket.addEventListener('open', () => {
        console.log('WebSocket connection opened');
        setIsConnected(true);
      });

      socket.addEventListener('error', (error) => {
        console.error('WebSocket error:', error);
        setIsConnected(false);
      });

      socket.addEventListener('close', () => {
        console.log('WebSocket connection closed');
        setIsConnected(false);
      });
    } else {
      console.error('Failed to initialize socket');
    }

    return () => {
      // Clean up socket connection when component unmounts
      if (socket) {
        console.log('Closing socket connection');
        socket.close();
      }
    };
  }, []);

  // Fetch contacts (followers/following)
  useEffect(() => {
    const fetchContacts = async () => {
      try {
        setIsLoading(true);

        // Get user's following list as contacts
        const response = await userAPI.getFollowing(user?.id);
        setContacts(response.data.data.following || []);
      } catch (error) {
        console.error('Error fetching contacts:', error);
      } finally {
        setIsLoading(false);
      }
    };

    if (user) {
      fetchContacts();
    }
  }, [user]);

  // Handle chat room subscription
  useEffect(() => {
    if (!selectedContact) return;

    // Create a room ID (combination of user IDs sorted alphabetically)
    const roomId = [user.id, selectedContact.id].sort().join('-');
    console.log("Joining chat room:", roomId);

    // Join the chat room
    joinChatRoom(roomId);

    // Subscribe to messages
    const unsubscribe = subscribeToMessages((data) => {
      console.log("Received WebSocket message:", data);
      console.log("Current room ID:", roomId);

      if (data.roomId === roomId) {
        console.log("Message is for current room");

        // Add the message to the chat (for both sent and received messages)
        // We'll handle deduplication by checking if the message already exists
        const newMessage = {
          content: data.message.content,
          sender: data.message.sender,
          timestamp: data.message.timestamp || new Date().toISOString(),
        };

        setMessages(prev => {
          // Check if message already exists to prevent duplicates
          const messageExists = prev.some(msg =>
            msg.content === newMessage.content &&
            msg.sender === newMessage.sender &&
            Math.abs(new Date(msg.timestamp) - new Date(newMessage.timestamp)) < 1000 // Within 1 second
          );

          if (!messageExists) {
            console.log("Adding new message:", newMessage);
            return [...prev, newMessage];
          } else {
            console.log("Message already exists, skipping");
            return prev;
          }
        });
      } else {
        console.log("Message is for different room, ignoring");
      }
    });

    // Fetch previous messages when joining a room
    const fetchMessages = async () => {
      try {
        console.log('Fetching messages for room:', roomId);
        const response = await messageAPI.getMessages(selectedContact.id);
        console.log('Loaded messages from API in room subscription:', response.data);

        if (response.data.success && response.data.messages) {
          // Convert messages to the format expected by the UI
          const formattedMessages = response.data.messages.map(msg => ({
            content: msg.content,
            sender: msg.senderId || msg.sender_id, // Handle both formats
            timestamp: msg.createdAt || msg.created_at, // Handle both formats
          }));
          // Reverse to show oldest first (API returns newest first)
          setMessages(formattedMessages.reverse());
          console.log('Formatted messages in room subscription:', formattedMessages);
        }
      } catch (error) {
        console.error('Error fetching messages in room subscription:', error);
        setMessages([]); // Clear messages on error
      }
    };

    fetchMessages();

    // Clean up on unmount or when changing contacts
    return () => {
      leaveChatRoom(roomId);
      unsubscribe();
    };
  }, [selectedContact, user]);

  // Scroll to bottom when messages change
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleContactSelect = async (contact) => {
    console.log("this is my contact", contact)
    setSelectedContact(contact);

    // Load existing messages with this contact
    try {
      const response = await messageAPI.getMessages(contact.id);
      console.log('Loaded messages from API:', response.data);

      if (response.data.success && response.data.messages) {
        // Convert messages to the format expected by the UI
        const formattedMessages = response.data.messages.map(msg => ({
          content: msg.content,
          sender: msg.senderId || msg.sender_id, // Handle both formats
          timestamp: msg.createdAt || msg.created_at, // Handle both formats
        }));
        // Reverse to show oldest first (API returns newest first)
        setMessages(formattedMessages.reverse());
        console.log('Formatted messages:', formattedMessages);
      }
    } catch (error) {
      console.error('Error loading messages:', error);
      setMessages([]); // Clear messages on error
    }
  };

  const handleSendMessage = async (e) => {
    e.preventDefault();

    if (!newMessage.trim() || !selectedContact) return;

    // Store message content and clear input immediately for better UX
    const messageContent = newMessage;
    setNewMessage('');

    try {
      // Send message to backend API
      console.log('Sending message to API:', {
        receiverId: selectedContact.id,
        content: messageContent,
      });

      const response = await messageAPI.sendMessage(selectedContact.id, messageContent);
      console.log('Message sent successfully to database:', response.data);

      // Note: We don't add the message to local state here because we'll receive it
      // via WebSocket broadcast, which ensures real-time delivery to all participants

    } catch (error) {
      console.error('Error sending message:', error);
      // Restore the message content to the input on error
      setNewMessage(messageContent);
      // Show error to user
      alert('Failed to send message. Please try again.');
    }
  };

  return (
    <ProtectedRoute>
      <div className={styles.chatContainer}>
        <div className={styles.chatSidebar}>
          <div className={styles.sidebarHeader}>
            <h2 className={styles.sidebarTitle}>Conversations</h2>
            <div className={`${styles.connectionStatus} ${isConnected ? styles.connected : styles.disconnected}`}>
              <span className={styles.statusDot}></span>
              {isConnected ? 'Online' : 'Offline'}
            </div>
          </div>

          {isLoading ? (
            <div className={styles.loading}>Loading contacts...</div>
          ) : contacts.length === 0 ? (
            <div className={styles.emptyContacts}>
              <p>No contacts found</p>
              <p>Follow people to start chatting with them</p>
            </div>
          ) : (
            <div className={styles.contactsList}>
              {contacts.map(contact => (
                <div
                  key={contact.id}
                  className={`${styles.contactItem} ${selectedContact?.id === contact.id ? styles.activeContact : ''}`}
                  onClick={() => handleContactSelect(contact)}
                >
                  {contact.profilePicture ? (
                    <Image
                      src={contact.profilePicture}
                      alt={contact.username}
                      width={40}
                      height={40}
                      className={styles.contactAvatar}
                    />
                  ) : (
                    <div className={styles.contactAvatarPlaceholder}>
                      {contact.username?.charAt(0).toUpperCase() || 'U'}
                    </div>
                  )}

                  <div className={styles.contactInfo}>
                    <h3 className={styles.contactName}>{contact.fullName}</h3>
                    <p className={styles.contactUsername}>@{contact.username}</p>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className={styles.chatMain}>
          {!selectedContact ? (
            <div className={styles.noChatSelected}>
              <p>Select a contact to start chatting</p>
            </div>
          ) : (
            <>
              <div className={styles.chatHeader}>
                <div className={styles.chatHeaderInfo}>
                  {selectedContact.profilePicture ? (
                    <Image
                      src={selectedContact.profilePicture}
                      alt={selectedContact.username}
                      width={40}
                      height={40}
                      className={styles.headerAvatar}
                    />
                  ) : (
                    <div className={styles.headerAvatarPlaceholder}>
                      {selectedContact.username?.charAt(0).toUpperCase() || 'U'}
                    </div>
                  )}

                  <div>
                    <h2 className={styles.headerName}>{selectedContact.fullName}</h2>
                    <p className={styles.headerUsername}>@{selectedContact.username}</p>
                  </div>
                </div>
              </div>

              <div className={styles.messagesContainer}>
                {messages.length === 0 ? (
                  <div className={styles.emptyMessages}>
                    <p>No messages yet</p>
                    <p>Send a message to start the conversation</p>
                  </div>
                ) : (
                  <div className={styles.messagesList}>
                    {messages.map((message, index) => (
                      <div
                        key={index}
                        className={`${styles.messageItem} ${message.sender === user.id ? styles.ownMessage : styles.otherMessage}`}
                      >
                        <div className={styles.messageContent}>
                          {message.content}
                        </div>
                        <div className={styles.messageTime}>
                          {new Date(message.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                        </div>
                      </div>
                    ))}
                    <div ref={messagesEndRef} />
                  </div>
                )}
              </div>

              <form onSubmit={handleSendMessage} className={styles.messageForm}>
                <input
                  type="text"
                  placeholder="Type a message..."
                  value={newMessage}
                  onChange={(e) => setNewMessage(e.target.value)}
                  className={styles.messageInput}
                />
                <Button
                  type="submit"
                  variant="primary"
                  disabled={!newMessage.trim()}
                >
                  Send
                </Button>
              </form>
            </>
          )}
        </div>
      </div>
    </ProtectedRoute>
  );
}
