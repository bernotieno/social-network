'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import Image from 'next/image';
import { useAuth } from '@/hooks/useAuth';
import { userAPI } from '@/utils/api';
import { getUserProfilePictureUrl, getFallbackAvatar } from '@/utils/images';
import Button from './Button';
import styles from '@/styles/UserCard.module.css';

export default function UserCard({ user, showFollowButton = true, onFollowChange }) {
  const { user: currentUser } = useAuth();
  const [isFollowing, setIsFollowing] = useState(user.isFollowing || false);
  const [isLoading, setIsLoading] = useState(false);

  const isOwnProfile = currentUser?.id === user.id;

  // Update follow state when user prop changes
  useEffect(() => {
    setIsFollowing(user.isFollowing || false);
  }, [user.isFollowing]);

  const handleFollow = async () => {
    if (isLoading || isOwnProfile) return;

    try {
      setIsLoading(true);

      if (isFollowing) {
        await userAPI.unfollow(user.id);
        setIsFollowing(false);
        if (onFollowChange) {
          onFollowChange(user.id, false);
        }
      } else {
        await userAPI.follow(user.id);
        setIsFollowing(true);
        if (onFollowChange) {
          onFollowChange(user.id, true);
        }
      }
    } catch (error) {
      console.error('Error following/unfollowing user:', error);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className={styles.userCard}>
      <Link href={`/profile/${user.id}`} className={styles.userLink}>
        <div className={styles.userAvatar}>
          {user.profilePicture ? (
            <img
              src={getUserProfilePictureUrl(user)}
              alt={user.username}
              className={styles.avatar}
              onError={(e) => {
                e.target.src = getFallbackAvatar(user);
              }}
            />
          ) : (
            <img
              src={getFallbackAvatar(user)}
              alt={user.username}
              className={styles.avatar}
            />
          )}
        </div>

        <div className={styles.userInfo}>
          <h3 className={styles.userName}>{user.fullName}</h3>
          <p className={styles.userUsername}>@{user.username}</p>
          {user.bio && (
            <p className={styles.userBio}>{user.bio}</p>
          )}
        </div>
      </Link>

      {showFollowButton && !isOwnProfile && (
        <div className={styles.userActions}>
          <Button
            variant={isFollowing ? 'secondary' : 'primary'}
            size="small"
            onClick={handleFollow}
            disabled={isLoading}
          >
            {isLoading ? 'Loading...' : (isFollowing ? 'Unfollow' : 'Follow')}
          </Button>
        </div>
      )}
    </div>
  );
}
