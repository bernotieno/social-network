'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import Image from 'next/image';
import { formatDistanceToNow } from 'date-fns';
import { useAuth } from '@/hooks/useAuth';
import { postAPI, groupAPI } from '@/utils/api';
import { getImageUrl, isGif, validateImageFile } from '@/utils/images';
import { subscribeToPostLikes, subscribeToNewComments, subscribeToCommentDeletions, subscribeToGroupPostLikes, subscribeToGroupPostComments, subscribeToGroupPostCommentDeletions } from '@/utils/socket';
import { useAlert } from '@/contexts/AlertContext';
import Button from '@/components/Button';
import { ConfirmModal, AlertModal } from '@/components/Modal';
import styles from '@/styles/Post.module.css';

const Post = ({ post, onDelete, onUpdate, isGroupPost = false, groupId = null, isGroupAdmin = false, groupCreatorId = null, priority = false }) => {
  const { user } = useAuth();
  const { showSuccess } = useAlert();

  // Early return if post is not properly loaded
  if (!post || !post.id) {
    return <div className={styles.post}>Loading post...</div>;
  }
  const [isLiked, setIsLiked] = useState(post.isLikedByCurrentUser || false);
  const [likesCount, setLikesCount] = useState(post.likesCount || 0);
  const [showComments, setShowComments] = useState(false);
  const [comments, setComments] = useState(post.comments || []);
  const [newComment, setNewComment] = useState('');
  const [commentImage, setCommentImage] = useState(null);
  const [commentImagePreview, setCommentImagePreview] = useState(null);
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

  /**
   * @summary Determines if the current user can delete the post.
   * @description A user can delete a post if:
   * - They are the author of the post (`isOwnPost`).
   * - It's a group post and the user is a group admin, but not the group creator.
   * - It's a group post and the user is the group creator (can delete any post in their group).
   * @summary Determines if the current user is the author of the post.
   * @description This variable is true if the logged-in user's ID matches the post's author ID.
   * @type {boolean}
   */
  const isOwnPost = user && user.id === post.userId;
  const canDeletePost = () => {
    if (!user || !post) return false;
    
    // No system-level admin permissions for post deletion
    
    // User can delete their own post
    if (user.id === post.userId) return true;

    // For group posts, check group-specific permissions
    if (isGroupPost && groupId) {
      // Only proceed if we have groupCreatorId
      if (!groupCreatorId) {
        return false;
      }
      
      // Group creator can delete any post in their group
      if (user.id === groupCreatorId) return true;
      
      // Group admin can delete member posts only, not creator posts
      // Make sure the current user is NOT the group creator and the post author is NOT the group creator
      if (isGroupAdmin && user.id !== groupCreatorId && post.userId !== groupCreatorId) {
        return true;
      }
    }

    return false;
  };
  const canEditPost = isOwnPost; // Only post author can edit

  const handleLikeToggle = async () => {
    // Store original state for potential rollback
    const originalIsLiked = isLiked;
    const originalLikesCount = likesCount;

    try {
      console.log('Like toggle clicked:', { isLiked, postId: post.id, isGroupPost, groupId });

      // Optimistic update
      const newIsLiked = !isLiked;
      const newLikesCount = newIsLiked ? likesCount + 1 : Math.max(0, likesCount - 1);

      setIsLiked(newIsLiked);
      setLikesCount(newLikesCount);

      if (isGroupPost && groupId) {
        // Use group post API
        if (originalIsLiked) {
          console.log('Unliking group post...');
          const response = await groupAPI.unlikeGroupPost(groupId, post.id);
          console.log('Unlike response:', response);
        } else {
          console.log('Liking group post...');
          const response = await groupAPI.likeGroupPost(groupId, post.id);
          console.log('Like response:', response);
        }
      } else {
        // Use regular post API
        if (originalIsLiked) {
          console.log('Unliking regular post...');
          const response = await postAPI.unlikePost(post.id);
          console.log('Unlike response:', response);
        } else {
          console.log('Liking regular post...');
          const response = await postAPI.likePost(post.id);
          console.log('Like response:', response);
        }
      }
    } catch (error) {
      console.error('Error toggling like:', error);
      console.error('Error details:', error.response?.data || error.message);

      // Revert the optimistic update on error
      setIsLiked(originalIsLiked);
      setLikesCount(originalLikesCount);

      // Show error message to user
      showAlert('Failed to update like. Please try again.', 'error');
    }
  };

  const handleCommentToggle = async () => {
    if (!showComments) {
      console.log(showComments)
      // Always fetch fresh comments when opening the comments section
      try {
        let response;
        if (isGroupPost && groupId) {
          response = await groupAPI.getGroupPostComments(groupId, post.id);
        } else {
          response = await postAPI.getComments(post.id);
        }
        console.log("Full response:", response)
        console.log("Response data:", response.data)
        console.log("Response data.data:", response.data?.data)

        // The API response structure is: { success: true, message: "...", data: { comments: [...] } }
        const comments = response.data?.data?.comments || [];
        setComments(comments);
        console.log("Comments:", comments)
      } catch (error) {
        console.error('Error fetching comments:', error);
        // Set empty array on error to show the "no comments" message
        setComments([]);
      }
    }
    setShowComments(!showComments);
  };

  const handleCommentImageChange = (e) => {
    const file = e.target.files[0];
    if (file) {
      // Validate file
      const validation = validateImageFile(file);
      if (!validation.isValid) {
        alert(validation.error);
        return;
      }

      setCommentImage(file);

      // Create preview URL
      const reader = new FileReader();
      reader.onloadend = () => {
        setCommentImagePreview(reader.result);
      };
      reader.readAsDataURL(file);
    }
  };

  const handleRemoveCommentImage = () => {
    setCommentImage(null);
    setCommentImagePreview(null);
  };

  const handleCommentSubmit = async (e) => {
    e.preventDefault();

    if (!newComment.trim() && !commentImage) return;

    try {
      setIsSubmittingComment(true);
      let response;

      if (commentImage) {
        // Use FormData for image upload
        const formData = new FormData();
        formData.append('content', newComment);
        formData.append('image', commentImage);

        if (isGroupPost && groupId) {
          response = await groupAPI.addGroupPostComment(groupId, post.id, formData);
        } else {
          response = await postAPI.addCommentWithImage(post.id, formData);
        }
      } else {
        // Use JSON for text-only comments
        if (isGroupPost && groupId) {
          response = await groupAPI.addGroupPostComment(groupId, post.id, newComment);
        } else {
          response = await postAPI.addComment(post.id, newComment);
        }
      }

      setComments(prev => [response.data.data?.comment, ...prev]);
      setNewComment('');
      setCommentImage(null);
      setCommentImagePreview(null);
    } catch (error) {
      console.error('Error adding comment:', error);
    } finally {
      setIsSubmittingComment(false);
    }
  };

  const handleDeleteComment = async (commentId) => {
    try {
      if (isGroupPost && groupId) {
        await groupAPI.deleteGroupPostComment(groupId, post.id, commentId);
      } else {
        await postAPI.deleteComment(post.id, commentId);
      }
      setComments(prev => prev.filter(comment => comment.id !== commentId));
    } catch (error) {
      console.error('Error deleting comment:', error);
    }
  };

  const handleDeletePost = () => {
    setShowDropdown(false); // Close dropdown
    setShowDeleteConfirm(true); // Show custom confirmation modal
  };

  /**
   * @summary Handles the deletion of a post.
   * @description This function sets the `isDeleting` state to true, calls the appropriate API to delete the post (either `postAPI.deletePost` for regular posts or `groupAPI.deleteGroupPost` for group posts), and then calls the `onDelete` callback if the deletion is successful. It also handles error logging.
   * @returns {void}
   */
  const confirmDeletePost = async () => {
    try {
      setIsDeleting(true);

      if (isGroupPost && groupId) {
        await groupAPI.deleteGroupPost(groupId, post.id);
      } else {
        await postAPI.deletePost(post.id);
      }

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
    let unsubscribeLikes, unsubscribeComments, unsubscribeCommentDeletions;

    if (isGroupPost) {
      // Subscribe to group post events
      unsubscribeLikes = subscribeToGroupPostLikes((data) => {
        if (post?.id && data.postId === post.id && data.groupId === groupId) {
          // Only update state if it's NOT the current user (to avoid double updates from optimistic updates)
          if (data.userId !== user?.id) {
            if (data.action === 'like') {
              setLikesCount(prev => prev + 1);
            } else if (data.action === 'unlike') {
              setLikesCount(prev => Math.max(0, prev - 1));
            }
          }
          // Note: We don't update isLiked state here for current user since optimistic update handles it
        }
      });

      unsubscribeComments = subscribeToGroupPostComments((data) => {
        if (post?.id && data.postId === post.id && data.groupId === groupId && showComments) {
          // Only add the comment if it's not from the current user (to avoid duplicates)
          // The current user's comments are already added optimistically
          if (data.comment?.author?.id !== user?.id) {
            setComments(prev => [...prev, data.comment]);
          }
        }
      });

      unsubscribeCommentDeletions = subscribeToGroupPostCommentDeletions((data) => {
        if (post?.id && data.postId === post.id && data.groupId === groupId && showComments) {
          setComments(prev => prev.filter(comment => comment.id !== data.commentId));
        }
      });
    } else {
      // Subscribe to regular post events
      unsubscribeLikes = subscribeToPostLikes((data) => {
        if (post?.id && data.postId === post.id) {
          // Only update state if it's NOT the current user (to avoid double updates from optimistic updates)
          if (data.userId !== user?.id) {
            if (data.action === 'like') {
              setLikesCount(prev => prev + 1);
            } else if (data.action === 'unlike') {
              setLikesCount(prev => Math.max(0, prev - 1));
            }
          }
          // Note: We don't update isLiked state here for current user since optimistic update handles it
        }
      });

      unsubscribeComments = subscribeToNewComments((data) => {
        if (post?.id && data.postId === post.id && showComments) {
          // Only add the comment if it's not from the current user (to avoid duplicates)
          // The current user's comments are already added optimistically
          if (data.comment?.author?.id !== user?.id) {
            setComments(prev => [...prev, data.comment]);
          }
        }
      });

      unsubscribeCommentDeletions = subscribeToCommentDeletions((data) => {
        if (post?.id && data.postId === post.id && showComments) {
          setComments(prev => prev.filter(comment => comment.id !== data.commentId));
        }
      });
    }

    return () => {
      if (unsubscribeLikes) unsubscribeLikes();
      if (unsubscribeComments) unsubscribeComments();
      if (unsubscribeCommentDeletions) unsubscribeCommentDeletions();
    };
  }, [post?.id, user?.id, showComments, isGroupPost, groupId]);

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

        {(canEditPost || canDeletePost()) && (
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
                {canEditPost && (
                  <button
                    className={styles.dropdownItem}
                    onClick={handleEditPost}
                  >
                    ✏️ Edit Post
                  </button>
                )}
                {canDeletePost() && (
                  <button
                    className={styles.dropdownItem}
                    onClick={handleDeletePost}
                  >
                    🗑️ Delete Post
                  </button>
                )}
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
                <option value="public">🌍 Public</option>
                <option value="followers">👥 Followers</option>
                {editVisibility === 'custom' && (
                  <option value="custom">🎯 Custom (current)</option>
                )}
                <option value="private">🔒 Private</option>
              </select>
              {editVisibility === 'custom' && (
                <div className={styles.customVisibilityNote}>
                  <small>Note: To change custom visibility settings, please create a new post.</small>
                </div>
              )}

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
                {isGif(post.image) ? (
                  // Use regular img tag for GIFs to preserve animation
                  <img
                    src={getImageUrl(post.image)}
                    alt="Post image"
                    className={styles.postImageElement}
                    style={{ width: '100%', height: 'auto', maxHeight: '500px', objectFit: 'contain' }}
                  />
                ) : (
                  // Use Next.js Image for static images
                  <Image
                    src={getImageUrl(post.image)}
                    alt="Post image"
                    fill
                    sizes="(max-width: 768px) 100vw, (max-width: 1200px) 50vw, 33vw"
                    priority={priority}
                    style={{ objectFit: 'cover' }}
                  />
                )}
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
              <div className={styles.commentInputContainer}>
                <input
                  type="text"
                  placeholder="Write a comment..."
                  value={newComment}
                  onChange={(e) => setNewComment(e.target.value)}
                  className={styles.commentInput}
                  disabled={isSubmittingComment}
                />
                <div className={styles.commentActions}>
                  <label className={styles.commentImageUpload}>
                    <span>📷</span>
                    <input
                      type="file"
                      accept="image/jpeg,image/jpg,image/png,image/gif"
                      onChange={handleCommentImageChange}
                      style={{ display: 'none' }}
                    />
                  </label>
                  <Button
                    type="submit"
                    variant="primary"
                    size="small"
                    disabled={isSubmittingComment || (!newComment.trim() && !commentImage)}
                  >
                    Post
                  </Button>
                </div>
              </div>

              {commentImagePreview && (
                <div className={styles.commentImagePreview}>
                  <img
                    src={commentImagePreview}
                    alt="Comment preview"
                    className={styles.commentPreviewImage}
                  />
                  <button
                    type="button"
                    className={styles.removeCommentImage}
                    onClick={handleRemoveCommentImage}
                    title="Remove image"
                  >
                    ✕
                  </button>
                </div>
              )}
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
                        {comment.content && (
                          <p className={styles.commentText}>{comment.content}</p>
                        )}
                        {comment.image && (
                          <div className={styles.commentImageContainer}>
                            {isGif(comment.image) ? (
                              <img
                                src={getImageUrl(comment.image)}
                                alt="Comment image"
                                className={styles.commentImage}
                              />
                            ) : (
                              <Image
                                src={getImageUrl(comment.image)}
                                alt="Comment image"
                                width={200}
                                height={150}
                                className={styles.commentImage}
                                style={{ objectFit: 'cover' }}
                              />
                            )}
                          </div>
                        )}
                      </div>

                      <div className={styles.commentMeta}>
                        <span className={styles.commentTime}>
                          {formatDistanceToNow(new Date(comment.createdAt), { addSuffix: true })}
                        </span>

                        {(user?.id === comment.author.id || isOwnPost || (isGroupPost && groupCreatorId && user?.id === groupCreatorId) || (isGroupPost && isGroupAdmin && user?.id !== groupCreatorId && comment.author.id !== groupCreatorId)) && (
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
