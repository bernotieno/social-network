'use client';

import { useState, useEffect } from 'react';
import { notificationAPI } from '@/utils/api';
import { useAuth } from './useAuth';

const useNotifications = () => {
  const [notifications, setNotifications] = useState([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const { isAuthenticated } = useAuth();

  // Fetch notifications on mount if authenticated
  useEffect(() => {
    if (isAuthenticated) {
      fetchNotifications();

      // TODO: Re-enable real-time notifications when WebSocket is fixed
      // const unsubscribe = subscribeToNotifications((notification) => {
      //   setNotifications(prev => [notification, ...prev]);
      //   setUnreadCount(prev => prev + 1);
      // });
      //
      // return () => {
      //   unsubscribe();
      // };
    }
  }, [isAuthenticated]);

  // Fetch notifications from API
  const fetchNotifications = async () => {
    try {
      setIsLoading(true);
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
      setIsLoading(false);
    }
  };

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
    fetchNotifications,
    markAsRead,
    markAllAsRead,
  };
};

export default useNotifications;
