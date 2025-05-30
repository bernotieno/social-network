'use client';

import { useEffect } from 'react';
import Link from 'next/link';
import Image from 'next/image';
import { formatDistanceToNow } from 'date-fns';
import useNotifications from '@/hooks/useNotifications';
import Button from '@/components/Button';
import ProtectedRoute from '@/components/ProtectedRoute';
import styles from '@/styles/Notifications.module.css';

export default function Notifications() {
  const {
    notifications,
    isLoading,
    isRefreshing,
    fetchNotifications,
    markAsRead,
    markAllAsRead
  } = useNotifications();

  useEffect(() => {
    fetchNotifications();
  }, [fetchNotifications]);

  const getNotificationContent = (notification) => {
    console.log("this is the notification", notification)
    switch (notification.type) {
      case 'follow_request':
        return (
          <>
            <span className={styles.notificationText}>
              requested to follow you
            </span>
            <div className={styles.notificationActions}>
              <Button
                variant="primary"
                size="small"
                onClick={() => handleFollowResponse(notification.id, true)}
              >
                Accept
              </Button>
              <Button
                variant="secondary"
                size="small"
                onClick={() => handleFollowResponse(notification.id, false)}
              >
                Decline
              </Button>
            </div>
          </>
        );
      case 'follow_accepted':
        return (
          <span className={styles.notificationText}>
            accepted your follow request
          </span>
        );
      case 'new_follower':
        return (
          <span className={styles.notificationText}>
            started following you
          </span>
        );
      case 'post_like':
        return (
          <span className={styles.notificationText}>
            liked your post
          </span>
        );
      case 'post_comment':
        return (
          <span className={styles.notificationText}>
            commented on your post: "{notification.data?.comment}"
          </span>
        );
      case 'group_invite':
        return (
          <>
            <span className={styles.notificationText}>
              invited you to join the group "{notification.data?.groupName}"
            </span>
            <div className={styles.notificationActions}>
              <Button
                variant="primary"
                size="small"
                onClick={() => handleGroupInviteResponse(notification.id, true)}
              >
                Join
              </Button>
              <Button
                variant="secondary"
                size="small"
                onClick={() => handleGroupInviteResponse(notification.id, false)}
              >
                Decline
              </Button>
            </div>
          </>
        );
      case 'group_join_request':
        return (
          <>
            <span className={styles.notificationText}>
              requested to join your group "{notification.data?.groupName}"
            </span>
            <div className={styles.notificationActions}>
              <Button
                variant="primary"
                size="small"
                onClick={() => handleGroupJoinResponse(notification.id, true)}
              >
                Accept
              </Button>
              <Button
                variant="secondary"
                size="small"
                onClick={() => handleGroupJoinResponse(notification.id, false)}
              >
                Decline
              </Button>
            </div>
          </>
        );
      case 'event_invite':
        return (
          <>
            <span className={styles.notificationText}>
              invited you to the event "{notification.data?.eventName}"
            </span>
            <div className={styles.notificationActions}>
              <Button
                variant="primary"
                size="small"
                onClick={() => handleEventResponse(notification.id, 'going')}
              >
                Going
              </Button>
              <Button
                variant="secondary"
                size="small"
                onClick={() => handleEventResponse(notification.id, 'maybe')}
              >
                Maybe
              </Button>
              <Button
                variant="outline"
                size="small"
                onClick={() => handleEventResponse(notification.id, 'not_going')}
              >
                Decline
              </Button>
            </div>
          </>
        );
      default:
        return (
          <span className={styles.notificationText}>
            {notification.content}
          </span>
        );
    }
  };

  // These functions would call the appropriate API endpoints
  const handleFollowResponse = (notificationId, accept) => {
    console.log(`Follow request ${accept ? 'accepted' : 'declined'}: ${notificationId}`);
    markAsRead(notificationId);
  };

  const handleGroupInviteResponse = (notificationId, accept) => {
    console.log(`Group invite ${accept ? 'accepted' : 'declined'}: ${notificationId}`);
    markAsRead(notificationId);
  };

  const handleGroupJoinResponse = (notificationId, accept) => {
    console.log(`Group join request ${accept ? 'accepted' : 'declined'}: ${notificationId}`);
    markAsRead(notificationId);
  };

  const handleEventResponse = (notificationId, response) => {
    console.log(`Event response ${response}: ${notificationId}`);
    markAsRead(notificationId);
  };

  return (
    <ProtectedRoute>
      <div className={styles.notificationsContainer}>
        <div className={styles.notificationsHeader}>
          <div className={styles.titleContainer}>
            <h1 className={styles.notificationsTitle}>Notifications</h1>
            {isRefreshing && (
              <div className={styles.refreshIndicator}>
                <div className={styles.spinner}></div>
                <span>Updating...</span>
              </div>
            )}
          </div>

          {notifications && notifications.length > 0 && (
            <Button
              variant="secondary"
              size="small"
              onClick={markAllAsRead}
            >
              Mark all as read
            </Button>
          )}
        </div>

        {isLoading ? (
          <div className={styles.loading}>Loading notifications...</div>
        ) : !notifications || notifications.length === 0 ? (
          <div className={styles.emptyNotifications}>
            <p>No notifications yet</p>
            <p>When you get notifications, they'll appear here</p>
          </div>
        ) : (
          <div className={styles.notificationsList}>
            {notifications.map(notification => (
              <div
                key={notification.id}
                className={`${styles.notificationItem} ${!notification.readAt ? styles.unread : ''}`}
                onClick={() => markAsRead(notification.id)}
              >
                <Link href={`/profile/${notification.sender.id}`} className={styles.notificationSender}>
                  {notification.sender.profilePicture ? (
                    <Image
                      src={notification.sender.profilePicture}
                      alt={notification.sender.username}
                      width={50}
                      height={50}
                      className={styles.senderAvatar}
                    />
                  ) : (
                    <div className={styles.senderAvatarPlaceholder}>
                      {notification.sender.username?.charAt(0).toUpperCase() || 'U'}
                    </div>
                  )}
                </Link>

                <div className={styles.notificationContent}>
                  <div className={styles.notificationHeader}>
                    <Link href={`/profile/${notification.sender.id}`} className={styles.senderName}>
                      {notification.sender.fullName}
                    </Link>
                    {getNotificationContent(notification)}
                  </div>

                  <div className={styles.notificationTime}>
                    {formatDistanceToNow(new Date(notification.createdAt), { addSuffix: true })}
                  </div>
                </div>

                {!notification.readAt && (
                  <div className={styles.unreadIndicator} />
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </ProtectedRoute>
  );
}
