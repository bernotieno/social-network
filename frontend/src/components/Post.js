'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import Image from 'next/image';
import { formatDistanceToNow } from 'date-fns';
import { useAuth } from '@/hooks/useAuth';
import { postAPI } from '@/utils/api';
import { getImageUrl } from '@/utils/images';
import { subscribeToPostLikes, subscribeToNewComments, subscribeToCommentDeletions } from '@/utils/socket';
import Button from '@/components/Button';
import { ConfirmModal, AlertModal } from '@/components/Modal';
import styles from '@/styles/Post.module.css';

const Post = ({ post, onDelete, onUpdate }) => {
  const { user } = useAuth();
  const [isLiked, setIsLiked] = useState(post.isLikedByCurrentUser || false);
  const [likesCount, setLikesCount] = useState(post.likesCount || 0);
  const [showComments, setShowComments] = useState(false);
  const [comments, setComments] = useState(post.comments || []);
  const [newComment, setNewComment] = useState('');
  const [isSubmittingComment, setIsSubmittingComment] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [editContent, setEditContent] = useState(post.content);
  const [editVisibility, setEditVisibility] = useState(post.visibility);
  const [isSubmittingEdit, setIsSubmittingEdit] = useState(false);
  const [currentContent, setCurrentContent] = useState(post.content);
  const [currentVisibility, setCurrentVisibility] = useState(post.visibility);
  const [showDropdown, setShowDropdown] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [showAlert, setShowAlert] = useState(false);
  const [alertConfig, setAlertConfig] = useState({ type: 'info', message: '', title: 'Alert' });
  const [isDeleting, setIsDeleting] = useState(false);

  const isOwnPost = user?.id === post.author.id;

  const handleLikeToggle = async () => {
    try {
      if (isLiked) {
        await postAPI.unlikePost(post.id);
        setIsLiked(false);
        setLikesCount(prev => prev - 1);
      } else {
        await postAPI.likePost(post.id);
        setIsLiked(true);
        setLikesCount(prev => prev + 1);
      }
    } catch (error) {
      console.error('Error toggling like:', error);
    }
  };

  const handleCommentToggle = async () => {
    if (!showComments) {
      console.log(showComments)
      // Always fetch fresh comments when opening the comments section
      try {
        const response = await postAPI.getComments(post.id);
        console.log(">>>>", response)
        setComments(response.data.data.comments || []);
        console.log(response.data.data.comments)
      } catch (error) {
        console.error('Error fetching comments:', error);
        // Set empty array on error to show the "no comments" message
        setComments([]);
      }
    }
    setShowComments(!showComments);
  };

  const handleCommentSubmit = async (e) => {
    e.preventDefault();

    if (!newComment.trim()) return;

    try {
      setIsSubmittingComment(true);
      const response = await postAPI.addComment(post.id, newComment);
      setComments(prev => [...prev, response.data.comment]);
      setNewComment('');
    } catch (error) {
      console.error('Error adding comment:', error);
    } finally {
      setIsSubmittingComment(false);
    }
  };

  const handleDeleteComment = async (commentId) => {
    try {
      await postAPI.deleteComment(post.id, commentId);
      setComments(prev => prev.filter(comment => comment.id !== commentId));
    } catch (error) {
      console.error('Error deleting comment:', error);
    }
  };

  const handleDeletePost = () => {
    setShowDropdown(false); // Close dropdown
    setShowDeleteConfirm(true); // Show custom confirmation modal
  };

  const confirmDeletePost = async () => {
    try {
      setIsDeleting(true);
      await postAPI.deletePost(post.id);
      if (onDelete) {
        onDelete(post.id);
      }
      setShowDeleteConfirm(false);
    } catch (error) {
      console.error('Error deleting post:', error);
      setShowDeleteConfirm(false);
      setAlertConfig({
        type: 'error',
        title: 'Delete Failed',
        message: 'Failed to delete post. Please try again.'
      });
      setShowAlert(true);
    } finally {
      setIsDeleting(false);
    }
  };

  // Close dropdown when clicking outside
  const handleClickOutside = (e) => {
    if (!e.target.closest('.post-dropdown')) {
      setShowDropdown(false);
    }
  };

  // Add click outside listener
  useEffect(() => {
    if (showDropdown) {
      document.addEventListener('click', handleClickOutside);
      return () => document.removeEventListener('click', handleClickOutside);
    }
  }, [showDropdown]);

  // Real-time WebSocket event listeners
  useEffect(() => {
    // Subscribe to post likes/unlikes
    const unsubscribeLikes = subscribeToPostLikes((data) => {
      if (data.postId === post.id) {
        if (data.action === 'like') {
          setLikesCount(prev => prev + 1);
          // If the current user liked it, update the like state
          if (data.userId === user?.id) {
            setIsLiked(true);
          }
        } else if (data.action === 'unlike') {
          setLikesCount(prev => Math.max(0, prev - 1));
          // If the current user unliked it, update the like state
          if (data.userId === user?.id) {
            setIsLiked(false);
          }
        }
      }
    });

    // Subscribe to new comments
    const unsubscribeComments = subscribeToNewComments((data) => {
      if (data.postId === post.id && showComments) {
        setComments(prev => [...prev, data.comment]);
      }
    });

    // Subscribe to comment deletions
    const unsubscribeCommentDeletions = subscribeToCommentDeletions((data) => {
      if (data.postId === post.id && showComments) {
        setComments(prev => prev.filter(comment => comment.id !== data.commentId));
      }
    });

    return () => {
      unsubscribeLikes();
      unsubscribeComments();
      unsubscribeCommentDeletions();
    };
  }, [post.id, user?.id, showComments]);

  const handleDropdownToggle = () => {
    setShowDropdown(!showDropdown);
  };

  const handleEditPost = () => {
    setIsEditing(true);
    setEditContent(currentContent);
    setEditVisibility(currentVisibility);
    setShowDropdown(false); // Close dropdown when editing
  };

  const handleCancelEdit = () => {
    setIsEditing(false);
    setEditContent(currentContent);
    setEditVisibility(currentVisibility);
  };

  const handleSaveEdit = async () => {
    if (!editContent.trim()) {
      setAlertConfig({
        type: 'warning',
        title: 'Invalid Content',
        message: 'Post content cannot be empty.'
      });
      setShowAlert(true);
      return;
    }

    try {
      setIsSubmittingEdit(true);
      const formData = new FormData();
      formData.append('content', editContent);
      formData.append('visibility', editVisibility);

      await postAPI.updatePost(post.id, formData);

      // Update the local state
      setCurrentContent(editContent);
      setCurrentVisibility(editVisibility);

      // Call onUpdate callback if provided
      if (onUpdate) {
        onUpdate(post.id, { content: editContent, visibility: editVisibility });
      }

      setIsEditing(false);

      // Show success message
      setAlertConfig({
        type: 'success',
        title: 'Post Updated',
        message: 'Your post has been updated successfully.'
      });
      setShowAlert(true);
    } catch (error) {
      console.error('Error updating post:', error);
      setAlertConfig({
        type: 'error',
        title: 'Update Failed',
        message: 'Failed to update post. Please try again.'
      });
      setShowAlert(true);
    } finally {
      setIsSubmittingEdit(false);
    }
  };

  return (
    <div className={styles.post}>
      <div className={styles.postHeader}>
        <Link href={`/profile/${post.author.id}`} className={styles.authorInfo}>
          {post.author.profilePicture ? (
            <Image
              src={getImageUrl(post.author.profilePicture)}
              alt={post.author.username}
              width={40}
              height={40}
              className={styles.authorAvatar}
            />
          ) : (
            <div className={styles.authorAvatarPlaceholder}>
              {post.author.username?.charAt(0).toUpperCase() || 'U'}
            </div>
          )}
          <div>
            <h3 className={styles.authorName}>{post.author.fullName}</h3>
            <p className={styles.postTime}>
              {formatDistanceToNow(new Date(post.createdAt), { addSuffix: true })}
            </p>
          </div>
        </Link>

        {isOwnPost && (
          <div className={`${styles.postActions} post-dropdown`}>
            <button
              className={styles.dropdownToggle}
              onClick={handleDropdownToggle}
              aria-label="Post options"
              disabled={isEditing}
            >
              ⋮
            </button>

            {showDropdown && (
              <div className={styles.dropdownMenu}>
                <button
                  className={styles.dropdownItem}
                  onClick={handleEditPost}
                >
                  ✏️ Edit Post
                </button>
                <button
                  className={styles.dropdownItem}
                  onClick={handleDeletePost}
                >
                  🗑️ Delete Post
                </button>
              </div>
            )}
          </div>
        )}
      </div>

      <div className={styles.postContent}>
        {isEditing ? (
          <div className={styles.editForm}>
            <textarea
              value={editContent}
              onChange={(e) => setEditContent(e.target.value)}
              className={styles.editTextarea}
              placeholder="What's on your mind?"
              disabled={isSubmittingEdit}
            />

            <div className={styles.editControls}>
              <select
                value={editVisibility}
                onChange={(e) => setEditVisibility(e.target.value)}
                className={styles.editVisibilitySelect}
                disabled={isSubmittingEdit}
              >
                <option value="public">Public</option>
                <option value="friends">Friends</option>
                <option value="private">Private</option>
              </select>

              <div className={styles.editButtons}>
                <Button
                  variant="secondary"
                  size="small"
                  onClick={handleCancelEdit}
                  disabled={isSubmittingEdit}
                >
                  Cancel
                </Button>
                <Button
                  variant="primary"
                  size="small"
                  onClick={handleSaveEdit}
                  disabled={isSubmittingEdit || !editContent.trim()}
                >
                  {isSubmittingEdit ? 'Saving...' : 'Save'}
                </Button>
              </div>
            </div>
          </div>
        ) : (
          <>
            <p className={styles.postText}>{currentContent}</p>
            {post.image && (
              <div className={styles.postImage}>
                <Image
                  src={getImageUrl(post.image)}
                  alt="Post image"
                  fill
                  style={{ objectFit: 'cover' }}
                />
              </div>
            )}
          </>
        )}
      </div>

      <div className={styles.postFooter}>
        <div className={styles.postStats}>
          {likesCount > 0 && (
            <span className={styles.likesCount}>
              {likesCount} {likesCount === 1 ? 'like' : 'likes'}
            </span>
          )}

          {post.commentsCount > 0 && (
            <span className={styles.commentsCount}>
              {post.commentsCount} {post.commentsCount === 1 ? 'comment' : 'comments'}
            </span>
          )}
        </div>

        <div className={styles.postButtons}>
          <button
            className={`${styles.postButton} ${isLiked ? styles.liked : ''}`}
            onClick={handleLikeToggle}
          >
            {isLiked ? '❤️' : '🤍'} Like
          </button>

          <button
            className={styles.postButton}
            onClick={handleCommentToggle}
          >
            💬 Comment
          </button>
        </div>

        {showComments && (
          <div className={styles.commentsSection}>
            <form onSubmit={handleCommentSubmit} className={styles.commentForm}>
              <input
                type="text"
                placeholder="Write a comment..."
                value={newComment}
                onChange={(e) => setNewComment(e.target.value)}
                className={styles.commentInput}
                disabled={isSubmittingComment}
              />
              <Button
                type="submit"
                variant="primary"
                size="small"
                disabled={isSubmittingComment || !newComment.trim()}
              >
                Post
              </Button>
            </form>

            <div className={styles.commentsList}>
              {comments.length === 0 ? (
                <p className={styles.noComments}>No comments yet. Be the first to comment!</p>
              ) : (
                comments.map(comment => (
                  <div key={comment.id} className={styles.comment}>
                    <Link href={`/profile/${comment.author.id}`} className={styles.commentAuthor}>
                      {comment.author.profilePicture ? (
                        <Image
                          src={getImageUrl(comment.author.profilePicture)}
                          alt={comment.author.username}
                          width={32}
                          height={32}
                          className={styles.commentAvatar}
                        />
                      ) : (
                        <div className={styles.commentAvatarPlaceholder}>
                          {comment.author.username?.charAt(0).toUpperCase() || 'U'}
                        </div>
                      )}
                    </Link>

                    <div className={styles.commentContent}>
                      <div className={styles.commentBubble}>
                        <h4 className={styles.commentAuthorName}>{comment.author.fullName}</h4>
                        <p className={styles.commentText}>{comment.content}</p>
                      </div>

                      <div className={styles.commentMeta}>
                        <span className={styles.commentTime}>
                          {formatDistanceToNow(new Date(comment.createdAt), { addSuffix: true })}
                        </span>

                        {(user?.id === comment.author.id || isOwnPost) && (
                          <button
                            className={styles.deleteCommentButton}
                            onClick={() => handleDeleteComment(comment.id)}
                            aria-label="Delete comment"
                          >
                            Delete
                          </button>
                        )}
                      </div>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>
        )}
      </div>

      {/* Delete Confirmation Modal */}
      <ConfirmModal
        isOpen={showDeleteConfirm}
        onClose={() => setShowDeleteConfirm(false)}
        onConfirm={confirmDeletePost}
        title="Delete Post"
        message="Are you sure you want to delete this post? This action cannot be undone."
        confirmText="Delete"
        cancelText="Cancel"
        confirmVariant="danger"
        isLoading={isDeleting}
      />

      {/* Alert Modal */}
      <AlertModal
        isOpen={showAlert}
        onClose={() => setShowAlert(false)}
        title={alertConfig.title}
        message={alertConfig.message}
        type={alertConfig.type}
      />
    </div>
  );
};

export default Post;
