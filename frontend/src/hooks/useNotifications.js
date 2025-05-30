'use client';

import { useState, useEffect, useCallback } from 'react';
import { notificationAPI } from '@/utils/api';
import { useAuth } from './useAuth';
import { initializeSocket, subscribeToNotifications } from '@/utils/socket';

const useNotifications = () => {
  const [notifications, setNotifications] = useState([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const { isAuthenticated } = useAuth();

  // Fetch notifications from API
  const fetchNotifications = useCallback(async (isAutoRefresh = false) => {
    try {
      if (isAutoRefresh) {
        setIsRefreshing(true);
      } else {
        setIsLoading(true);
      }

      const response = await notificationAPI.getNotifications();
      console.log(">>>>res2", response)
      // Handle case where notifications might be null
      const notificationsData = response.data.data.notifications || [];
      setNotifications(notificationsData);

      // Calculate unread count
      const unread = notificationsData.filter(
        notification => !notification.readAt
      ).length;

      setUnreadCount(unread);
    } catch (error) {
      console.error('Error fetching notifications:', error);
      // Set empty array on error to prevent undefined issues
      setNotifications([]);
      setUnreadCount(0);
    } finally {
      if (isAutoRefresh) {
        setIsRefreshing(false);
      } else {
        setIsLoading(false);
      }
    }
  }, []);

  // Fetch notifications on mount if authenticated
  useEffect(() => {
    if (isAuthenticated) {
      fetchNotifications();

      // Initialize WebSocket for real-time notifications
      let unsubscribe;
      let pollingInterval;

      try {
        // Try to initialize WebSocket
        const socket = initializeSocket();
        if (socket) {
          console.log('WebSocket initialized successfully for notifications');
          unsubscribe = subscribeToNotifications((notification) => {
            console.log('Received real-time notification:', notification);
            setNotifications(prev => [notification, ...prev]);
            setUnreadCount(prev => prev + 1);
          });
        } else {
          console.warn('WebSocket initialization returned null');
        }
      } catch (error) {
        console.warn('WebSocket initialization failed, falling back to polling:', error);
      }

      // Auto-refresh: Poll for new notifications every 10 seconds
      // Only poll when page is visible to save resources
      const startPolling = () => {
        pollingInterval = setInterval(() => {
          if (!document.hidden) {
            fetchNotifications(true); // true indicates this is an auto-refresh
          }
        }, 10000);
      };

      const handleVisibilityChange = () => {
        if (document.hidden) {
          // Page is hidden, clear polling
          if (pollingInterval) {
            clearInterval(pollingInterval);
            pollingInterval = null;
          }
        } else {
          // Page is visible, restart polling
          if (!pollingInterval) {
            startPolling();
          }
        }
      };

      // Start initial polling
      startPolling();

      // Listen for page visibility changes
      document.addEventListener('visibilitychange', handleVisibilityChange);

      return () => {
        if (unsubscribe) {
          unsubscribe();
        }
        if (pollingInterval) {
          clearInterval(pollingInterval);
        }
        document.removeEventListener('visibilitychange', handleVisibilityChange);
      };
    }
  }, [isAuthenticated, fetchNotifications]);

  // Mark notification as read
  const markAsRead = async (notificationId) => {
    try {
      await notificationAPI.markAsRead(notificationId);

      // Update local state
      setNotifications(prev =>
        prev.map(notification =>
          notification.id === notificationId
            ? { ...notification, readAt: new Date().toISOString() }
            : notification
        )
      );

      setUnreadCount(prev => Math.max(0, prev - 1));
    } catch (error) {
      console.error('Error marking notification as read:', error);
    }
  };

  // Mark all notifications as read
  const markAllAsRead = async () => {
    try {
      await notificationAPI.markAllAsRead();

      // Update local state
      setNotifications(prev =>
        prev.map(notification => ({
          ...notification,
          readAt: notification.readAt || new Date().toISOString()
        }))
      );

      setUnreadCount(0);
    } catch (error) {
      console.error('Error marking all notifications as read:', error);
    }
  };

  return {
    notifications,
    unreadCount,
    isLoading,
    isRefreshing,
    fetchNotifications,
    markAsRead,
    markAllAsRead,
  };
};

export default useNotifications;
