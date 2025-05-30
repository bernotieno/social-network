let socket = null;
let eventListeners = new Map();

/**
 * Initialize WebSocket connection
 * @returns {WebSocket} - Native WebSocket instance
 */
export const initializeSocket = () => {
  if (!socket || socket.readyState === WebSocket.CLOSED) {
    // For browser clients, we'll rely on session cookies instead of tokens
    // The WebSocket will authenticate using the same session cookie as API calls
    const wsUrl = process.env.NEXT_PUBLIC_SOCKET_URL || 'ws://localhost:8080';
    const url = `${wsUrl}/ws`;
    console.log('WebSocket URL:', url);

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
 * @param {string} message - Message content
 */
export const sendMessage = (roomId, message) => {
  emit('send_message', { roomId, message });
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
