'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import Image from 'next/image';
import { useAuth } from '@/hooks/useAuth';
import { postAPI } from '@/utils/api';
import Button from '@/components/Button';
import Post from '@/components/Post';
import styles from '@/styles/Home.module.css';

export default function Home() {
  const { isAuthenticated, user } = useAuth();
  const [posts, setPosts] = useState([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    // Fetch feed posts if user is authenticated
    if (isAuthenticated) {
      fetchFeed();
    } else {
      setIsLoading(false);
    }
  }, [isAuthenticated]);

  const fetchFeed = async () => {
    try {
      setIsLoading(true);
      const response = await postAPI.getFeed();
      setPosts(response.data.data.posts || []);
    } catch (error) {
      console.error('Error fetching feed:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleDeletePost = (postId) => {
    setPosts(prev => prev.filter(post => post.id !== postId));
  };

  // Guest home page
  if (!isAuthenticated) {
    return (
      <div className={styles.guestContainer}>
        <div className={styles.heroSection}>
          <h1 className={styles.heroTitle}>Connect with friends and the world around you</h1>
          <p className={styles.heroSubtitle}>
            Join our social network to share, connect, and discover
          </p>
          <div className={styles.heroCta}>
            <Link href="/auth/register">
              <Button variant="primary" size="large">
                Sign Up
              </Button>
            </Link>
            <Link href="/auth/login">
              <Button variant="outline" size="large">
                Log In
              </Button>
            </Link>
          </div>
        </div>
        <div className={styles.featuresSection}>
          <div className={styles.feature}>
            <div className={styles.featureIcon}>👥</div>
            <h2>Connect with Friends</h2>
            <p>Follow friends and family to stay updated with their lives</p>
          </div>
          <div className={styles.feature}>
            <div className={styles.featureIcon}>📝</div>
            <h2>Share Your Thoughts</h2>
            <p>Post updates, photos, and more to share with your network</p>
          </div>
          <div className={styles.feature}>
            <div className={styles.featureIcon}>💬</div>
            <h2>Real-time Chat</h2>
            <p>Message friends instantly with our real-time chat system</p>
          </div>
          <div className={styles.feature}>
            <div className={styles.featureIcon}>👥</div>
            <h2>Join Groups</h2>
            <p>Find communities of people who share your interests</p>
          </div>
        </div>
      </div>
    );
  }

  // Authenticated home page (feed)
  return (
    <div className={styles.feedContainer}>
      {isLoading ? (
        <div className={styles.loading}>Loading feed...</div>
      ) : posts.length === 0 ? (
        <div className={styles.emptyFeed}>
          <h2>Your feed is empty</h2>
          <p>Follow more people or create a post to see content here</p>
          <Link href="/posts/create">
            <Button variant="primary">Create a Post</Button>
          </Link>
        </div>
      ) : (
        <div className={styles.feedContent}>
          <div className={styles.feedHeader}>
            <h1 className={styles.feedTitle}>Your Feed</h1>
            <Link href="/posts/create">
              <Button variant="primary">Create Post</Button>
            </Link>
          </div>

          <div className={styles.postsContainer}>
            {posts.map(post => (
              <Post
                key={post.id}
                post={post}
                onDelete={handleDeletePost}
              />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
