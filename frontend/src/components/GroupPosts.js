'use client';

import { useState, useEffect } from 'react';
import Image from 'next/image';
import { useAuth } from '@/hooks/useAuth';
import { groupAPI } from '@/utils/api';
import { getUserProfilePictureUrl, getFallbackAvatar, isGif, validateImageFile } from '@/utils/images';
import { useAlert } from '@/contexts/AlertContext';
import Button from '@/components/Button';
import Post from '@/components/Post';
import FollowerSelector from '@/components/FollowerSelector';
import SelectedFollowersTags from '@/components/SelectedFollowersTags';
import styles from '@/styles/GroupPosts.module.css';

export default function GroupPosts({ groupId, isGroupMember, isGroupAdmin, groupCreatorId }) {
  const { user } = useAuth();
  const { showSuccess, showError } = useAlert();
  const [posts, setPosts] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isCreating, setIsCreating] = useState(false);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [newPost, setNewPost] = useState({
    content: '',
    image: null
  });
  const [imagePreview, setImagePreview] = useState(null);

  useEffect(() => {
    fetchPosts();
  }, [groupId]);

  const fetchPosts = async () => {
    try {
      setIsLoading(true);
      const response = await groupAPI.getGroupPosts(groupId);

      if (response.data.success) {
        setPosts(response.data.data.posts || []);
      }
    } catch (error) {
      console.error('Error fetching group posts:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleImageChange = (e) => {
    const file = e.target.files[0];
    if (file) {
      // Validate file using utility function
      const validation = validateImageFile(file);
      if (!validation.isValid) {
        showError(validation.error, 'Invalid File');
        return;
      }

      setNewPost(prev => ({ ...prev, image: file }));
      const reader = new FileReader();
      reader.onload = (e) => {
        setImagePreview(e.target.result);
      };
      reader.readAsDataURL(file);
    }
  };

  const handleCreatePost = async (e) => {
    e.preventDefault();

    if (!newPost.content.trim() && !newPost.image) {
      showError('Please add some content or an image', 'Missing Content');
      return;
    }

    setIsCreating(true);

    try {
      const formData = new FormData();
      formData.append('content', newPost.content);

      if (newPost.image) {
        formData.append('image', newPost.image);
      }

      const response = await groupAPI.createGroupPost(groupId, formData);

      if (response.data.success) {
        // Add new post to the beginning of the list
        const newPostData = response.data.data.post;
        setPosts(prev => [newPostData, ...prev]);

        // Show success message
        showSuccess('Your post has been shared with the group!', 'Post Created');

        // Reset form
        setNewPost({ content: '', image: null });
        setImagePreview(null);
        setShowCreateForm(false);
      }
    } catch (error) {
      console.error('Error creating post:', error);
      const errorMessage = error.response?.data?.message || 'Failed to create post. Please try again.';
      showError(errorMessage, 'Failed to Create Post');
    } finally {
      setIsCreating(false);
    }
  };

  const handlePostUpdate = (updatedPost) => {
    setPosts(prev => prev.map(post =>
      post.id === updatedPost.id ? updatedPost : post
    ));
  };

  const handlePostDelete = (deletedPostId) => {
    setPosts(prev => prev.filter(post => post.id !== deletedPostId));
  };

  if (isLoading) {
    return <div className={styles.loading}>Loading posts...</div>;
  }

  return (
    <div className={styles.groupPostsContainer}>
      {/* Create Post Form */}
      {isGroupMember && (
        <div className={styles.createPostSection}>
          {!showCreateForm ? (
            <div className={styles.createPostPrompt} onClick={() => setShowCreateForm(true)}>
              <div className={styles.userAvatar}>
                {user?.profilePicture ? (
                  <Image
                    src={getUserProfilePictureUrl(user)}
                    alt={user.username}
                    width={40}
                    height={40}
                    className={styles.avatar}
                    onError={(e) => {
                      e.target.src = getFallbackAvatar(user);
                    }}
                  />
                ) : (
                  <Image
                    src={getFallbackAvatar(user)}
                    alt={user.username}
                    width={40}
                    height={40}
                    className={styles.avatar}
                  />
                )}
              </div>
              <div className={styles.promptText}>
                What&apos;s on your mind, {user?.fullName?.split(' ')[0] || user?.username}?
              </div>
            </div>
          ) : (
            <form onSubmit={handleCreatePost} className={styles.createPostForm}>
              <div className={styles.formHeader}>
                <div className={styles.userInfo}>
                  {user?.profilePicture ? (
                    <Image
                      src={getUserProfilePictureUrl(user)}
                      alt={user.username}
                      width={40}
                      height={40}
                      className={styles.avatar}
                      onError={(e) => {
                        e.target.src = getFallbackAvatar(user);
                      }}
                    />
                  ) : (
                    <Image
                      src={getFallbackAvatar(user)}
                      alt={user.username}
                      width={40}
                      height={40}
                      className={styles.avatar}
                    />
                  )}
                  <span className={styles.userName}>{user?.fullName || user?.username}</span>
                </div>
                <button
                  type="button"
                  className={styles.closeButton}
                  onClick={() => {
                    setShowCreateForm(false);
                    setNewPost({ content: '', image: null });
                    setImagePreview(null);
                  }}
                >
                  ✕
                </button>
              </div>

              <textarea
                placeholder="What's happening in the group?"
                value={newPost.content}
                onChange={(e) => setNewPost(prev => ({ ...prev, content: e.target.value }))}
                className={styles.contentInput}
                rows={3}
              />

              {imagePreview && (
                <div className={styles.imagePreview}>
                  {newPost.image && isGif(newPost.image.name) ? (
                    // Use regular img tag for GIFs to preserve animation
                    <img
                      src={imagePreview}
                      alt="Preview"
                      style={{
                        width: '200px',
                        height: '200px',
                        objectFit: 'cover',
                        borderRadius: '8px'
                      }}
                    />
                  ) : (
                    // Use Next.js Image for static images
                    <Image
                      src={imagePreview}
                      alt="Preview"
                      width={200}
                      height={200}
                      style={{ objectFit: 'cover' }}
                    />
                  )}
                  <button
                    type="button"
                    className={styles.removeImage}
                    onClick={() => {
                      setNewPost(prev => ({ ...prev, image: null }));
                      setImagePreview(null);
                    }}
                  >
                    ✕
                  </button>
                </div>
              )}

              <div className={styles.formActions}>
                <div className={styles.attachments}>
                  <input
                    type="file"
                    accept="image/jpeg,image/jpg,image/png,image/gif"
                    onChange={handleImageChange}
                    className={styles.fileInput}
                    id="postImage"
                  />
                  <label htmlFor="postImage" className={styles.attachButton}>
                    📷 Photo/GIF
                  </label>
                </div>

                <Button
                  type="submit"
                  variant="primary"
                  disabled={isCreating || (!newPost.content.trim() && !newPost.image)}
                >
                  {isCreating ? 'Posting...' : 'Post'}
                </Button>
              </div>
            </form>
          )}
        </div>
      )}

      {/* Posts List */}
      <div className={styles.postsList}>
        {posts.length === 0 ? (
          <div className={styles.emptyPosts}>
            <p>No posts yet</p>
            {isGroupMember && (
              <p>Be the first to share something with the group!</p>
            )}
          </div>
        ) : (
          posts.map(post => (
            <div key={post.id} className={styles.postWrapper}>
              <Post
              key={post.id}
              post={post}
              onUpdate={handlePostUpdate}
              onDelete={handlePostDelete}
              isGroupPost={true}
              groupId={groupId}
              isGroupAdmin={isGroupAdmin}
              groupCreatorId={groupCreatorId}
            />
            </div>
          ))
        )}
      </div>
    </div>
  );
}
