import { getToken } from './auth';

let socket = null;
let eventListeners = new Map();

/**
 * Initialize WebSocket connection
 * @returns {WebSocket} - Native WebSocket instance
 */
export const initializeSocket = () => {
  if (!socket || socket.readyState === WebSocket.CLOSED) {
    // Get the authentication token
    const token = getToken();
    console.log('Retrieved token for WebSocket:', token ? 'Token found' : 'No token');
    if (!token) {
      console.warn('No authentication token found, cannot initialize WebSocket');
      return null;
    }

    // Include token as query parameter for WebSocket authentication
    const wsUrl = process.env.NEXT_PUBLIC_SOCKET_URL || 'ws://localhost:8080';
    const url = `${wsUrl}/ws?token=${encodeURIComponent(token)}`;
    console.log('Initializing WebSocket with URL:', url);

    try {
      socket = new WebSocket(url);

      // Socket event listeners
      socket.onopen = () => {
        console.log('Socket connected');
        // Trigger custom 'connect' event
        triggerEvent('connect');
      };

      socket.onclose = () => {
        console.log('Socket disconnected');
        // Trigger custom 'disconnect' event
        triggerEvent('disconnect');
      };

      socket.onerror = (error) => {
        console.error('Socket connection error:', error);
        console.error('Socket connection error details:', JSON.stringify(error, Object.getOwnPropertyNames(error)));
        // Trigger custom 'connect_error' event
        triggerEvent('connect_error', error);
      };

      socket.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          // Trigger custom event based on message type
          if (data.type) {
            triggerEvent(data.type, data.payload);
          }
        } catch (error) {
          console.error('Error parsing WebSocket message:', error);
        }
      };
    } catch (error) {
      console.error('Error creating WebSocket connection:', error);
      return null;
    }
  }

  return socket;
};

/**
 * Trigger custom events
 * @param {string} eventName - Event name
 * @param {any} data - Event data
 */
const triggerEvent = (eventName, data) => {
  const listeners = eventListeners.get(eventName) || [];
  listeners.forEach(callback => callback(data));
};

/**
 * Add event listener (Socket.IO-like API)
 * @param {string} eventName - Event name
 * @param {function} callback - Callback function
 */
export const on = (eventName, callback) => {
  if (!eventListeners.has(eventName)) {
    eventListeners.set(eventName, []);
  }
  eventListeners.get(eventName).push(callback);
};

/**
 * Remove event listener (Socket.IO-like API)
 * @param {string} eventName - Event name
 * @param {function} callback - Callback function
 */
export const off = (eventName, callback) => {
  const listeners = eventListeners.get(eventName) || [];
  const index = listeners.indexOf(callback);
  if (index > -1) {
    listeners.splice(index, 1);
  }
};

/**
 * Emit event to server (Socket.IO-like API)
 * @param {string} eventName - Event name
 * @param {any} data - Data to send
 */
export const emit = (eventName, data) => {
  if (socket && socket.readyState === WebSocket.OPEN) {
    socket.send(JSON.stringify({
      type: eventName,
      payload: data
    }));
  }
};

/**
 * Get the socket instance
 * @returns {WebSocket|null} - WebSocket instance or null if not initialized
 */
export const getSocket = () => socket;

/**
 * Disconnect the socket
 */
export const disconnectSocket = () => {
  if (socket) {
    socket.close();
    socket = null;
    eventListeners.clear();
  }
};

/**
 * Subscribe to a chat room
 * @param {string} roomId - Room ID to join
 */
export const joinChatRoom = (roomId) => {
  emit('join_room', { roomId });
};

/**
 * Leave a chat room
 * @param {string} roomId - Room ID to leave
 */
export const leaveChatRoom = (roomId) => {
  emit('leave_room', { roomId });
};

/**
 * Send a message to a chat room
 * @param {string} roomId - Room ID to send message to
 * @param {object} message - Message content
 */
export const sendMessage = (roomId, message) => {
  emit('chat_message', { roomId, content: message });
};

/**
 * Subscribe to new messages in a chat room
 * @param {function} callback - Function to call when a new message is received
 * @returns {function} - Function to unsubscribe
 */
export const subscribeToMessages = (callback) => {
  on('new_message', callback);
  return () => off('new_message', callback);
};

/**
 * Subscribe to notifications
 * @param {function} callback - Function to call when a new notification is received
 * @returns {function} - Function to unsubscribe
 */
export const subscribeToNotifications = (callback) => {
  on('notification', callback);
  return () => off('notification', callback);
};

/**
 * Subscribe to post likes/unlikes
 * @param {function} callback - Function to call when a post is liked/unliked
 * @returns {function} - Function to unsubscribe
 */
export const subscribeToPostLikes = (callback) => {
  on('post_like', callback);
  return () => off('post_like', callback);
};

/**
 * Subscribe to new posts
 * @param {function} callback - Function to call when a new post is created
 * @returns {function} - Function to unsubscribe
 */
export const subscribeToNewPosts = (callback) => {
  on('new_post', callback);
  return () => off('new_post', callback);
};

/**
 * Subscribe to new comments
 * @param {function} callback - Function to call when a new comment is added
 * @returns {function} - Function to unsubscribe
 */
export const subscribeToNewComments = (callback) => {
  on('new_comment', callback);
  return () => off('new_comment', callback);
};

/**
 * Subscribe to user presence updates (online/offline)
 * @param {function} callback - Function to call when user presence changes
 * @returns {function} - Function to unsubscribe
 */
export const subscribeToUserPresence = (callback) => {
  on('user_presence', callback);
  return () => off('user_presence', callback);
};

/**
 * Subscribe to comment deletions
 * @param {function} callback - Function to call when a comment is deleted
 * @returns {function} - Function to unsubscribe
 */
export const subscribeToCommentDeletions = (callback) => {
  on('comment_deleted', callback);
  return () => off('comment_deleted', callback);
};
