'use client';

import { useState, useEffect, useRef } from 'react';
import Image from 'next/image';
import { useAuth } from '@/hooks/useAuth';
import { userAPI, messageAPI } from '@/utils/api';
import {
  initializeSocket,
  joinChatRoom,
  leaveChatRoom,
  subscribeToMessages,
  subscribeToUserPresence,
  subscribeToTypingStatus,
  sendTypingStatus
} from '@/utils/socket';
import { getUserProfilePictureUrl, getFallbackAvatar } from '@/utils/images';
import { useAlert } from '@/contexts/AlertContext';
import Button from '@/components/Button';
import ProtectedRoute from '@/components/ProtectedRoute';
import styles from '@/styles/Chat.module.css';
import emojis from '@/components/emojis'

export default function Chat() {
  const { user } = useAuth();
  const { showError } = useAlert();
  const [contacts, setContacts] = useState([]);
  const [selectedContact, setSelectedContact] = useState(null);
  const [messages, setMessages] = useState([]);
  const [newMessage, setNewMessage] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [onlineUsers, setOnlineUsers] = useState(new Set());
  const [typingUsers, setTypingUsers] = useState(new Map()); // Map of roomId -> Set of userIds
  const [isTyping, setIsTyping] = useState(false);
  const [currentRoomId, setCurrentRoomId] = useState(null); // Track which room the user is currently in

  const messagesEndRef = useRef(null);
  const typingTimeoutRef = useRef(null);

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  // Helper function to check if a user is typing in a specific room
  const isUserTypingInRoom = (userId, roomId) => {
    const roomTypingUsers = typingUsers.get(roomId);
    return roomTypingUsers ? roomTypingUsers.has(userId) : false;
  };

  // Helper function to get room ID for a contact
  const getRoomId = (contactId) => {
    return [user.id, contactId].sort().join('-');
  };

  // Helper function to determine where to show typing indicator
  const shouldShowTypingInHeader = (contactId) => {
    const roomId = getRoomId(contactId);
    // Show in header only if:
    // 1. Current user is in this room (selectedContact matches)
    // 2. The contact is typing in this room
    return selectedContact?.id === contactId &&
           currentRoomId === roomId &&
           isUserTypingInRoom(contactId, roomId);
  };

  const shouldShowTypingInContact = (contactId) => {
    const roomId = getRoomId(contactId);
    // Show in contact list if:
    // 1. Current user is NOT in this room (either no room selected or different room)
    // 2. The contact is typing in this room
    return currentRoomId !== roomId &&
           isUserTypingInRoom(contactId, roomId);
  };

  // Initialize socket connection
  useEffect(() => {
    console.log('Initializing socket connection...');
    const socket = initializeSocket();
    if (socket) {
      console.log('Socket initialized successfully');

      // Add connection event listeners
      socket.addEventListener('open', () => {
        console.log('WebSocket connection opened');
      });

      socket.addEventListener('error', (error) => {
        console.error('WebSocket error:', error);
      });

      socket.addEventListener('close', () => {
        console.log('WebSocket connection closed');
        setOnlineUsers(new Set()); // Clear online users when disconnected
      });

      // Subscribe to user presence updates
      const unsubscribePresence = subscribeToUserPresence((presenceData) => {
        console.log('User presence update:', presenceData);
        if (presenceData.userId && presenceData.status) {
          setOnlineUsers(prev => {
            const newSet = new Set(prev);
            if (presenceData.status === 'online') {
              newSet.add(presenceData.userId);
            } else {
              newSet.delete(presenceData.userId);
            }
            return newSet;
          });
        }
      });

      // Subscribe to typing status updates
      const unsubscribeTyping = subscribeToTypingStatus((typingData) => {
        console.log('Typing status update:', typingData);
        if (typingData.userId && typingData.roomId) {
          setTypingUsers(prev => {
            const newMap = new Map(prev);
            const roomTypingUsers = newMap.get(typingData.roomId) || new Set();

            if (typingData.isTyping) {
              roomTypingUsers.add(typingData.userId);
            } else {
              roomTypingUsers.delete(typingData.userId);
            }

            if (roomTypingUsers.size > 0) {
              newMap.set(typingData.roomId, roomTypingUsers);
            } else {
              newMap.delete(typingData.roomId);
            }

            return newMap;
          });
        }
      });

      return () => {
        unsubscribePresence();
        unsubscribeTyping();
      };
    } else {
      console.error('Failed to initialize socket');
    }

    return () => {
      // Clean up socket connection when component unmounts
      if (socket) {
        console.log('Closing socket connection');
        socket.close();
      }

      // Clear typing timeout
      if (typingTimeoutRef.current) {
        clearTimeout(typingTimeoutRef.current);
      }
    };
  }, []);

  // Fetch contacts (followers and following) and online users
  useEffect(() => {
    const fetchContactsAndOnlineUsers = async () => {
      try {
        setIsLoading(true);

        // Get both followers and following for chat contacts, and online users
        const [followingResponse, followersResponse, onlineUsersResponse] = await Promise.all([
          userAPI.getFollowing(user?.id),
          userAPI.getFollowers(user?.id),
          messageAPI.getOnlineUsers()
        ]);

        const following = followingResponse.data.data?.following || [];
        const followers = followersResponse.data.data?.followers || [];
        const onlineUsersData = onlineUsersResponse.data.onlineUsers || [];

        // Combine and deduplicate contacts
        const contactsMap = new Map();

        // Add following users
        following.forEach(user => {
          contactsMap.set(user.id, user);
        });

        // Add followers
        followers.forEach(user => {
          contactsMap.set(user.id, user);
        });

        // Convert map to array
        setContacts(Array.from(contactsMap.values()));

        // Set initial online users
        const onlineUserIds = new Set(onlineUsersData.map(user => user.id));
        setOnlineUsers(onlineUserIds);
      } catch (error) {
        console.error('Error fetching contacts and online users:', error);
      } finally {
        setIsLoading(false);
      }
    };

    if (user) {
      fetchContactsAndOnlineUsers();
    }
  }, [user]);

  // Handle chat room subscription
  useEffect(() => {
    if (!selectedContact) {
      setCurrentRoomId(null);
      return;
    }

    // Create a room ID (combination of user IDs sorted alphabetically)
    const roomId = [user.id, selectedContact.id].sort().join('-');
    console.log("Joining chat room:", roomId);

    // Set current room ID
    setCurrentRoomId(roomId);

    // Join the chat room
    joinChatRoom(roomId);

    // Subscribe to messages
    const unsubscribe = subscribeToMessages((data) => {
      console.log("Received WebSocket message:", data);
      console.log("Current room ID:", roomId);

      // Extract payload from WebSocket message
      const payload = data.payload || data;
      const messageRoomId = payload.roomId || data.roomId;
      const messageData = payload.message || data.message;

      if (messageRoomId === roomId) {
        console.log("Message is for current room");

        // Add the message to the chat (for both sent and received messages)
        // We'll handle deduplication by checking if the message already exists
        const newMessage = {
          content: messageData.content,
          sender: messageData.sender,
          timestamp: messageData.timestamp || new Date().toISOString(),
        };

        setMessages(prev => {
          console.log("Processing WebSocket message:", newMessage);
          console.log("Current messages:", prev);

          // Check if message already exists to prevent duplicates
          // First check for exact optimistic message match
          const optimisticMessageIndex = prev.findIndex(msg => {
            const isMatch = msg.isOptimistic &&
              msg.content === newMessage.content &&
              msg.sender === newMessage.sender;
            console.log("Checking optimistic message:", msg, "Match:", isMatch);
            return isMatch;
          });

          if (optimisticMessageIndex !== -1) {
            // Replace optimistic message with confirmed message
            console.log("Replacing optimistic message with confirmed message at index:", optimisticMessageIndex);
            const updatedMessages = [...prev];
            updatedMessages[optimisticMessageIndex] = { ...newMessage, isOptimistic: false };
            console.log("Updated messages:", updatedMessages);
            return updatedMessages;
          }

          // Check if message already exists (for regular duplicates)
          const messageExists = prev.some(msg =>
            !msg.isOptimistic &&
            msg.content === newMessage.content &&
            msg.sender === newMessage.sender &&
            Math.abs(new Date(msg.timestamp) - new Date(newMessage.timestamp)) < 5000 // Within 5 seconds
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

    // Clear typing users when switching contacts
    setTypingUsers(new Map());

    // Clear current room (will be set in the room subscription effect)
    setCurrentRoomId(null);

    // Stop current typing status
    if (isTyping) {
      setIsTyping(false);
      if (typingTimeoutRef.current) {
        clearTimeout(typingTimeoutRef.current);
      }
    }

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

  // Handle emoji insertion
  const insertEmoji = (emoji) => {
    setNewMessage(prev => prev + emoji);
  };

  // Handle typing status
  const handleTyping = (value) => {
    setNewMessage(value);

    if (!selectedContact) return;

    const roomId = [user.id, selectedContact.id].sort().join('-');

    if (value.trim() && !isTyping) {
      // User started typing
      setIsTyping(true);
      sendTypingStatus(roomId, true);
    } else if (!value.trim() && isTyping) {
      // User stopped typing (cleared input)
      setIsTyping(false);
      sendTypingStatus(roomId, false);
    }

    // Clear existing timeout
    if (typingTimeoutRef.current) {
      clearTimeout(typingTimeoutRef.current);
    }

    // Set timeout to stop typing indicator after 3 seconds of inactivity
    if (value.trim()) {
      typingTimeoutRef.current = setTimeout(() => {
        setIsTyping(false);
        sendTypingStatus(roomId, false);
      }, 3000);
    }
  };

  const handleSendMessage = async (e) => {
    e.preventDefault();

    if (!newMessage.trim() || !selectedContact) return;

    // Store message content and clear input immediately for better UX
    const messageContent = newMessage;
    setNewMessage('');

    // Stop typing indicator when sending message
    if (isTyping) {
      setIsTyping(false);
      const roomId = [user.id, selectedContact.id].sort().join('-');
      sendTypingStatus(roomId, false);
    }

    // Clear typing timeout
    if (typingTimeoutRef.current) {
      clearTimeout(typingTimeoutRef.current);
    }

    // Create optimistic message object for immediate display
    const optimisticMessage = {
      content: messageContent,
      sender: user.id,
      timestamp: new Date().toISOString(),
      isOptimistic: true, // Flag to identify optimistic messages
    };

    console.log("Creating optimistic message:", optimisticMessage);

    // Add message to local state immediately for instant display
    setMessages(prevMessages => {
      // console.log("Adding optimistic message to state:", prevMessages);
      return [...prevMessages, optimisticMessage];
    });

    try {
      // Send message to backend API
      console.log('Sending message to API:', {
        receiverId: selectedContact.id,
        content: messageContent,
      });

      const response = await messageAPI.sendMessage(selectedContact.id, messageContent);
      console.log('Message sent successfully to database:', response.data);

      // Fallback: Remove optimistic flag after successful send
      // This ensures the "Sending..." indicator disappears even if WebSocket doesn't work
      setTimeout(() => {
        setMessages(prevMessages =>
          prevMessages.map(msg =>
            msg.isOptimistic && msg.content === messageContent && msg.sender === user.id
              ? { ...msg, isOptimistic: false }
              : msg
          )
        );
      }, 500); // Wait 500ms to allow WebSocket to handle it first

    } catch (error) {
      console.error('Error sending message:', error);

      // Remove the optimistic message on error
      setMessages(prevMessages =>
        prevMessages.filter(msg =>
          !(msg.isOptimistic && msg.content === messageContent && msg.sender === user.id)
        )
      );

      // Restore the message content to the input on error
      setNewMessage(messageContent);
      // Show error to user
      showError('Failed to send message. Please try again.');
    }
  };

  const emojiList = emojis();
  // const emojiList = ['😀', '😂', '😍', '🤔', '👍', '👎', '❤️', '🎉', '🔥', '💯',"👷🏿","🖐","🤚🏼","🖐","🤚🏻","🤚","💋","❤‍🔥","😈","🤯","🧐","🤮","🤔","🤒","🤚🏾","🤛","👊","✊🏿","👊🏽","🤝🏼","💅","👩🏻‍🍳",
  //   "🇰🇪",];
  const [showEmojis, setShowEmojis] = useState(false);

  const handleEmojiClick = (emoji) => {
    handleTyping(newMessage + emoji); // Append emoji to message
  };


  return (
    <ProtectedRoute>
      <div className={styles.chatContainer}>
        <div className={styles.chatSidebar}>
          <div className={styles.sidebarHeader}>
            <h2 className={styles.sidebarTitle}>Conversations</h2>
          </div>

          {isLoading ? (
            <div className={styles.loading}>Loading contacts...</div>
          ) : contacts.length === 0 ? (
            <div className={styles.emptyContacts}>
              <p>No contacts found</p>
              <p>Follow people or get followers to start chatting</p>
            </div>
          ) : (
            <div className={styles.contactsList}>
              {contacts.map(contact => (
                <div
                  key={contact.id}
                  className={`${styles.contactItem} ${selectedContact?.id === contact.id ? styles.activeContact : ''}`}
                  onClick={() => handleContactSelect(contact)}
                >
                  <div className={styles.contactAvatarContainer}>
                    {contact.profilePicture ? (
                      <Image
                        src={getUserProfilePictureUrl(contact)}
                        alt={contact.username}
                        width={40}
                        height={40}
                        className={styles.contactAvatar}
                        onError={(e) => {
                          e.target.src = getFallbackAvatar(contact);
                        }}
                      />
                    ) : (
                      <Image
                        src={getFallbackAvatar(contact)}
                        alt={contact.username}
                        width={40}
                        height={40}
                        className={styles.contactAvatar}
                      />
                    )}
                    {onlineUsers.has(contact.id) && (
                      <div className={styles.onlineIndicator}></div>
                    )}
                  </div>

                  <div className={styles.contactInfo}>
                    <h3 className={styles.contactName}>{contact.fullName}</h3>
                    <p className={styles.contactUsername}>
                      @{contact.username}
                      {shouldShowTypingInContact(contact.id) && (
                        <span className={styles.typingText}> • typing...</span>
                      )}
                    </p>
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
                  <div className={styles.contactAvatarContainer}>
                    {selectedContact.profilePicture ? (
                      <Image
                        src={getUserProfilePictureUrl(selectedContact)}
                        alt={selectedContact.username}
                        width={40}
                        height={40}
                        className={styles.headerAvatar}
                        onError={(e) => {
                          e.target.src = getFallbackAvatar(selectedContact);
                        }}
                      />
                    ) : (
                      <Image
                        src={getFallbackAvatar(selectedContact)}
                        alt={selectedContact.username}
                        width={40}
                        height={40}
                        className={styles.headerAvatar}
                      />
                    )}
                    {onlineUsers.has(selectedContact.id) && (
                      <div className={styles.onlineIndicator}></div>
                    )}
                  </div>

                  <div>
                    <h2 className={styles.headerName}>{selectedContact.fullName}</h2>
                    <p className={styles.headerUsername}>
                      @{selectedContact.username}
                      {(() => {
                        if (shouldShowTypingInHeader(selectedContact.id)) {
                          return <span className={styles.typingText}> • typing...</span>;
                        } else if (onlineUsers.has(selectedContact.id)) {
                          return <span className={styles.onlineText}> • Online</span>;
                        }
                        return null;
                      })()}
                    </p>
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
                        className={`${styles.messageItem} ${message.sender === user.id ? styles.ownMessage : styles.otherMessage} ${message.isOptimistic ? styles.optimisticMessage : ''}`}
                      >
                        <div className={styles.messageContent}>
                          {message.content}
                          {message.isOptimistic && (
                            <span className={styles.sendingIndicator}>Sending...</span>
                          )}
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

                {showEmojis && (
                    <div className={styles.emojiBar}>
                      {emojiList.map((emoji,key) => (
                          <button
                              key={key}
                              type="button"
                              className={styles.emojiButton}
                              onClick={() => handleEmojiClick(emoji)}
                          >
                            {emoji}
                          </button>
                      ))}
                    </div>
                )}

                <div className={styles.inputContainer}>
                  <input
                    type="text"
                    placeholder="Type a message..."
                    value={newMessage}
                    onChange={(e) => handleTyping(e.target.value)}
                    className={styles.messageInput}
                  />

                  {/*toggle emoji box*/}
                  <button
                      type="button"
                      onClick={() => setShowEmojis(!showEmojis)}
                      className={styles.emojiToggleButton}
                  >
                    {showEmojis ? 'hide emojis' : 'show more emojis'}
                  </button>

                  <Button
                    type="submit"
                    variant="primary"
                    disabled={!newMessage.trim()}
                  >
                    Send
                  </Button>
                </div>
              </form>
            </>
          )}
        </div>
      </div>
    </ProtectedRoute>
  );
}
