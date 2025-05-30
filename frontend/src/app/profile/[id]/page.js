'use client';

import { useState, useEffect } from 'react';
import Image from 'next/image';
import { useParams } from 'next/navigation';
import { useAuth } from '@/hooks/useAuth';
import { userAPI, postAPI } from '@/utils/api';
import { getUserProfilePictureUrl, getUserCoverPhotoUrl, getFallbackAvatar } from '@/utils/images';
import Button from '@/components/Button';
import Post from '@/components/Post';
import UserCard from '@/components/UserCard';
import ProtectedRoute from '@/components/ProtectedRoute';
import styles from '@/styles/Profile.module.css';

export default function ProfilePage() {
  const { id } = useParams();
  const { user: currentUser, updateUserData } = useAuth();

  const [profile, setProfile] = useState(null);
  const [posts, setPosts] = useState([]);
  const [followers, setFollowers] = useState([]);
  const [following, setFollowing] = useState([]);
  const [activeTab, setActiveTab] = useState('posts');
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingFollowers, setIsLoadingFollowers] = useState(false);
  const [isLoadingFollowing, setIsLoadingFollowing] = useState(false);
  const [isFollowing, setIsFollowing] = useState(false);
  const [followersCount, setFollowersCount] = useState(0);
  const [followingCount, setFollowingCount] = useState(0);
  const [showEditModal, setShowEditModal] = useState(false);
  const [editFormData, setEditFormData] = useState({
    fullName: '',
    bio: '',
    isPrivate: false
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [profilePicFile, setProfilePicFile] = useState(null);
  const [coverPhotoFile, setCoverPhotoFile] = useState(null);
  const [profilePicPreview, setProfilePicPreview] = useState(null);
  const [coverPhotoPreview, setCoverPhotoPreview] = useState(null);
  const [updateMessage, setUpdateMessage] = useState(null);

  const isOwnProfile = currentUser?.id === id;

  useEffect(() => {
    fetchProfileData();
  }, [id]);

  useEffect(() => {
    if (profile && profile.user) {
      setEditFormData({
        fullName: profile.user.fullName || '',
        bio: profile.user.bio || '',
        isPrivate: profile.user.isPrivate || false
      });
    }
  }, [profile]);

  // Remove automatic upload - we'll handle it in the form submission
  // useEffect(() => {
  //   if (profilePicFile && isOwnProfile) {
  //     // Use a try-catch block to prevent unhandled promise rejections
  //     (async () => {
  //       try {
  //         await uploadProfilePic();
  //       } catch (error) {
  //         console.error('Failed to upload profile picture:', error);
  //         // Reset the file state on error
  //         setProfilePicFile(null);
  //       }
  //     })();
  //   }
  // }, [profilePicFile, isOwnProfile]);

  const fetchProfileData = async () => {
    try {
      setIsLoading(true);

      // Fetch user profile
      const profileResponse = await userAPI.getProfile(id);
      setProfile(profileResponse.data.data);

      // Check if current user is following this profile
      if (currentUser && !isOwnProfile) {
        setIsFollowing(profileResponse.data.data.isFollowedByCurrentUser || false);
      }

      // Set followers and following counts
      setFollowersCount(profileResponse.data.data.followersCount || 0);
      setFollowingCount(profileResponse.data.data.followingCount || 0);

      // Fetch user posts
      const postsResponse = await postAPI.getPosts(id);
      setPosts(postsResponse.data.data.posts || []);
    } catch (error) {
      console.error('Error fetching profile data:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleFollow = async () => {
    try {
      if (isFollowing) {
        await userAPI.unfollow(id);
        setIsFollowing(false);
        setFollowersCount(prev => prev - 1);

        // Remove current user from followers list if it's loaded
        if (followers.length > 0) {
          setFollowers(prev => prev.filter(follower => follower.id !== currentUser?.id));
        }
      } else {
        await userAPI.follow(id);
        setIsFollowing(true);
        setFollowersCount(prev => prev + 1);

        // Add current user to followers list if it's loaded
        if (followers.length > 0 && currentUser) {
          setFollowers(prev => [currentUser, ...prev]);
        }
      }
    } catch (error) {
      console.error('Error following/unfollowing user:', error);
    }
  };

  const fetchFollowers = async () => {
    try {
      setIsLoadingFollowers(true);
      const response = await userAPI.getFollowers(id);
      setFollowers(response.data.data.followers || []);
    } catch (error) {
      console.error('Error fetching followers:', error);
      setFollowers([]);
    } finally {
      setIsLoadingFollowers(false);
    }
  };

  const fetchFollowing = async () => {
    try {
      setIsLoadingFollowing(true);
      const response = await userAPI.getFollowing(id);
      setFollowing(response.data.data.following || []);
    } catch (error) {
      console.error('Error fetching following:', error);
      setFollowing([]);
    } finally {
      setIsLoadingFollowing(false);
    }
  };

  const handleTabChange = (tab) => {
    setActiveTab(tab);

    // Fetch data when switching to followers/following tabs
    if (tab === 'followers' && followers.length === 0) {
      fetchFollowers();
    } else if (tab === 'following' && following.length === 0) {
      fetchFollowing();
    }
  };

  const handleEditProfile = () => {
    setShowEditModal(true);
  };

  const handleCloseModal = () => {
    setShowEditModal(false);
  };

  const handleEditFormChange = (e) => {
    const { name, value, type, checked } = e.target;
    setEditFormData(prev => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value
    }));
  };

  const handleProfilePicChange = (e) => {
    const file = e.target.files[0];
    if (file) {
      setProfilePicFile(file);
      const reader = new FileReader();
      reader.onloadend = () => {
        setProfilePicPreview(reader.result);
      };
      reader.readAsDataURL(file);
    }
  };

  const handleCoverPhotoChange = (e) => {
    const file = e.target.files[0];
    if (file) {
      setCoverPhotoFile(file);
      const reader = new FileReader();
      reader.onloadend = () => {
        setCoverPhotoPreview(reader.result);
      };
      reader.readAsDataURL(file);
    }
  };

  const uploadProfilePic = async () => {
    if (!profilePicFile) return;

    try {
      const formData = new FormData();
      formData.append('avatar', profilePicFile);

      console.log('Uploading profile picture...');
      const response = await userAPI.uploadAvatar(formData);
      console.log('Profile picture upload response:', response);

      if (response.data && response.data.data && response.data.data.user) {
        // Update profile with new profile picture URL
        const updatedUser = response.data.data.user;
        setProfile(prev => ({
          ...prev,
          user: {
            ...prev.user,
            profilePicture: updatedUser.profilePicture
          }
        }));

        // Update auth context
        updateUserData({ profilePicture: updatedUser.profilePicture });
      } else {
        console.error('Invalid response format from avatar upload:', response);
      }

      // Reset file state
      setProfilePicFile(null);
      setProfilePicPreview(null);
    } catch (error) {
      console.error('Error uploading profile picture:', error);
      // Keep the preview if there was an error
      setProfilePicFile(null);
    }
  };

  const uploadCoverPhoto = async () => {
    if (!coverPhotoFile) return;

    try {
      const formData = new FormData();
      formData.append('coverPhoto', coverPhotoFile);

      console.log('Uploading cover photo...');
      const response = await userAPI.uploadCoverPhoto(formData);
      console.log('Cover photo upload response:', response);

      if (response.data && response.data.data && response.data.data.user) {
        // Update profile with new cover photo URL
        const updatedUser = response.data.data.user;
        setProfile(prev => ({
          ...prev,
          user: {
            ...prev.user,
            coverPhoto: updatedUser.coverPhoto
          }
        }));

        // Update auth context
        updateUserData({ coverPhoto: updatedUser.coverPhoto });
      } else {
        console.error('Invalid response format from cover photo upload:', response);
      }

      // Reset file state
      setCoverPhotoFile(null);
      setCoverPhotoPreview(null);
    } catch (error) {
      console.error('Error uploading cover photo:', error);
      // Keep the preview if there was an error
      setCoverPhotoFile(null);
    }
  };

  const handleSubmitEdit = async (e) => {
    e.preventDefault();
    setIsSubmitting(true);

    try {
      // Update profile data
      const response = await userAPI.updateProfile(editFormData);
      console.log('Profile update response:', response);

      // Upload profile picture if selected
      if (profilePicFile) {
        await uploadProfilePic();
      }

      // Upload cover photo if selected
      if (coverPhotoFile) {
        await uploadCoverPhoto();
      }

      // Update local state with the response from the server
      if (response.data && response.data.data && response.data.data.user) {
        const updatedUser = response.data.data.user;
        setProfile(prev => ({
          ...prev,
          user: updatedUser
        }));

        // Update auth context with the updated user data
        updateUserData(updatedUser);
      } else {
        // Fallback: update with form data if response structure is unexpected
        setProfile(prev => ({
          ...prev,
          user: {
            ...prev.user,
            ...editFormData
          }
        }));

        // Update auth context
        updateUserData(editFormData);
      }

      // Close modal
      setShowEditModal(false);

      // Show success message
      setUpdateMessage({ type: 'success', text: 'Profile updated successfully!' });
      setTimeout(() => setUpdateMessage(null), 3000);

    } catch (error) {
      console.error('Error updating profile:', error);
      // Show error message to user
      setUpdateMessage({ type: 'error', text: 'Failed to update profile. Please try again.' });
      setTimeout(() => setUpdateMessage(null), 5000);
    } finally {
      setIsSubmitting(false);
    }
  };

  if (isLoading) {
    return (
      <div className={styles.loading}>
        Loading profile...
      </div>
    );
  }

  console.log("this is the profile", profile);

  if (!profile) {
    return (
      <div className={styles.notFound}>
        <h1>Profile not found</h1>
        <p>The user you're looking for doesn't exist or has been removed.</p>
      </div>
    );
  }

  return (
    <ProtectedRoute>
      <div className={styles.profileContainer}>
        {updateMessage && (
          <div className={`${styles.updateMessage} ${styles[updateMessage.type]}`}>
            {updateMessage.text}
          </div>
        )}
        <div className={styles.profileHeader}>
          <div className={styles.profileCover}>
            {profile.user.coverPhoto || coverPhotoPreview ? (
              <img
                src={coverPhotoPreview || getUserCoverPhotoUrl(profile.user)}
                alt={`${profile.user.username}'s cover`}
                style={{
                  width: '100%',
                  height: '100%',
                  objectFit: 'cover',
                  position: 'absolute',
                  top: 0,
                  left: 0
                }}
                onError={(e) => {
                  console.error('Cover photo failed to load:', e.target.src);
                }}
              />
            ) : (
              <div className={styles.defaultCover} />
            )}
            {isOwnProfile && (
              <label className={styles.coverPhotoUpload} title="Change cover photo">
                <span>📷</span>
                <input
                  type="file"
                  accept="image/*"
                  onChange={handleCoverPhotoChange}
                  style={{ display: 'none' }}
                />
              </label>
            )}
          </div>

          <div className={styles.profileInfo}>
            <div className={styles.profilePicture}>
              {profile.user.profilePicture || profilePicPreview ? (
                <img
                  src={profilePicPreview || getUserProfilePictureUrl(profile.user)}
                  alt={profile.user.username}
                  className={styles.avatar}
                  onError={(e) => {
                    console.error('Profile picture failed to load:', e.target.src);
                  }}
                />
              ) : (
                <img
                  src={getFallbackAvatar(profile.user)}
                  alt={profile.user.username}
                  className={styles.avatar}
                />
              )}
              {isOwnProfile && (
                <label className={styles.profilePicUpload} title="Change profile picture">
                  <span>📷</span>
                  <input
                    type="file"
                    accept="image/*"
                    onChange={handleProfilePicChange}
                    style={{ display: 'none' }}
                  />
                </label>
              )}
            </div>

            <div className={styles.profileDetails}>
              <h1 className={styles.profileName}>{profile.user.fullName}</h1>
              <p className={styles.profileUsername}>@{profile.user.username}</p>

              {profile.user.bio && (
                <p className={styles.profileBio}>{profile.user.bio}</p>
              )}

              <div className={styles.profileStats}>
                <div className={styles.stat}>
                  <span className={styles.statCount}>{posts.length}</span>
                  <span className={styles.statLabel}>Posts</span>
                </div>
                <div className={styles.stat}>
                  <span className={styles.statCount}>{followersCount}</span>
                  <span className={styles.statLabel}>Followers</span>
                </div>
                <div className={styles.stat}>
                  <span className={styles.statCount}>{followingCount}</span>
                  <span className={styles.statLabel}>Following</span>
                </div>
              </div>
            </div>

            <div className={styles.profileActions}>
              {isOwnProfile ? (
                <Button variant="outline" onClick={handleEditProfile}>Edit Profile</Button>
              ) : (
                <Button
                  variant={isFollowing ? 'secondary' : 'primary'}
                  onClick={handleFollow}
                >
                  {isFollowing ? 'Unfollow' : 'Follow'}
                </Button>
              )}
            </div>
          </div>
        </div>

        <div className={styles.profileContent}>
          <div className={styles.profileTabs}>
            <button
              className={`${styles.tabButton} ${activeTab === 'posts' ? styles.activeTab : ''}`}
              onClick={() => handleTabChange('posts')}
            >
              Posts
            </button>
            <button
              className={`${styles.tabButton} ${activeTab === 'followers' ? styles.activeTab : ''}`}
              onClick={() => handleTabChange('followers')}
            >
              Followers
            </button>
            <button
              className={`${styles.tabButton} ${activeTab === 'following' ? styles.activeTab : ''}`}
              onClick={() => handleTabChange('following')}
            >
              Following
            </button>
          </div>

          <div className={styles.tabContent}>
            {activeTab === 'posts' && (
              <div className={styles.postsGrid}>
                {posts.length === 0 ? (
                  <div className={styles.emptyState}>
                    <p>No posts yet</p>
                    {isOwnProfile && (
                      <p>Share your first post to get started!</p>
                    )}
                  </div>
                ) : (
                  <div className={styles.postsList}>
                    {posts.map(post => (
                      <Post
                        key={post.id}
                        post={post}
                        onDelete={(postId) => {
                          setPosts(prev => prev.filter(p => p.id !== postId));
                        }}
                        onUpdate={(postId, updatedData) => {
                          setPosts(prev => prev.map(p =>
                            p.id === postId ? { ...p, ...updatedData } : p
                          ));
                        }}
                      />
                    ))}
                  </div>
                )}
              </div>
            )}

            {activeTab === 'followers' && (
              <div className={styles.followersGrid}>
                {isLoadingFollowers ? (
                  <div className={styles.loading}>
                    <p>Loading followers...</p>
                  </div>
                ) : followers.length === 0 ? (
                  <div className={styles.emptyState}>
                    <p>No followers yet</p>
                  </div>
                ) : (
                  <div className={styles.usersList}>
                    {followers.map(follower => (
                      <UserCard
                        key={follower.id}
                        user={follower}
                        showFollowButton={follower.id !== currentUser?.id}
                        onFollowChange={(userId, isFollowing) => {
                          // Update local state if needed
                          console.log(`User ${userId} follow status changed to ${isFollowing}`);
                          // Update the follower's isFollowing status in the local state
                          setFollowers(prev => prev.map(f =>
                            f.id === userId ? { ...f, isFollowing } : f
                          ));
                        }}
                      />
                    ))}
                  </div>
                )}
              </div>
            )}

            {activeTab === 'following' && (
              <div className={styles.followingGrid}>
                {isLoadingFollowing ? (
                  <div className={styles.loading}>
                    <p>Loading following...</p>
                  </div>
                ) : following.length === 0 ? (
                  <div className={styles.emptyState}>
                    <p>Not following anyone yet</p>
                  </div>
                ) : (
                  <div className={styles.usersList}>
                    {following.map(followedUser => (
                      <UserCard
                        key={followedUser.id}
                        user={followedUser}
                        showFollowButton={followedUser.id !== currentUser?.id}
                        onFollowChange={(userId, isFollowing) => {
                          // Update local state if needed
                          if (!isFollowing) {
                            // Remove from following list if unfollowed
                            setFollowing(prev => prev.filter(user => user.id !== userId));
                            setFollowingCount(prev => prev - 1);
                          }
                          // Update the followed user's isFollowing status in the local state
                          setFollowing(prev => prev.map(f =>
                            f.id === userId ? { ...f, isFollowing } : f
                          ));
                        }}
                      />
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      </div>

      {showEditModal && (
        <div className={styles.modalOverlay}>
          <div className={styles.modalContent}>
            <h2>Edit Profile</h2>
            <form onSubmit={handleSubmitEdit}>
              <div className={styles.formGroup}>
                <label htmlFor="fullName">Full Name</label>
                <input
                  type="text"
                  id="fullName"
                  name="fullName"
                  value={editFormData.fullName}
                  onChange={handleEditFormChange}
                  className={styles.input}
                />
              </div>

              <div className={styles.formGroup}>
                <label htmlFor="bio">Bio</label>
                <textarea
                  id="bio"
                  name="bio"
                  value={editFormData.bio}
                  onChange={handleEditFormChange}
                  className={styles.textarea}
                  rows={4}
                />
              </div>

              <div className={styles.formGroup}>
                <label htmlFor="profilePic">Profile Picture</label>
                <input
                  type="file"
                  id="profilePic"
                  accept="image/*"
                  onChange={handleProfilePicChange}
                  className={styles.fileInput}
                />
                {profilePicPreview && (
                  <div className={styles.imagePreview}>
                    <img src={profilePicPreview} alt="Profile preview" />
                  </div>
                )}
              </div>

              <div className={styles.formGroup}>
                <label htmlFor="coverPhoto">Cover Photo</label>
                <input
                  type="file"
                  id="coverPhoto"
                  accept="image/*"
                  onChange={handleCoverPhotoChange}
                  className={styles.fileInput}
                />
                {coverPhotoPreview && (
                  <div className={styles.coverImagePreview}>
                    <img src={coverPhotoPreview} alt="Cover preview" />
                  </div>
                )}
              </div>

              <div className={styles.formGroup}>
                <label className={styles.checkboxLabel}>
                  <input
                    type="checkbox"
                    name="isPrivate"
                    checked={editFormData.isPrivate}
                    onChange={handleEditFormChange}
                  />
                  Private Account
                </label>
              </div>

              <div className={styles.modalActions}>
                <Button
                  type="button"
                  variant="secondary"
                  onClick={handleCloseModal}
                >
                  Cancel
                </Button>
                <Button
                  type="submit"
                  variant="primary"
                  disabled={isSubmitting}
                >
                  {isSubmitting ? 'Saving...' : 'Save Changes'}
                </Button>
              </div>
            </form>
          </div>
        </div>
      )}
    </ProtectedRoute>
  );
}
