'use client';

import { useState, useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Image from 'next/image';
import { useAuth } from '@/hooks/useAuth';
import { groupAPI } from '@/utils/api';
import { getImageUrl } from '@/utils/images';
import { useAlert } from '@/contexts/AlertContext';
import Button from '@/components/Button';
import ProtectedRoute from '@/components/ProtectedRoute';
import GroupPosts from '@/components/GroupPosts';
import GroupEvents from '@/components/GroupEvents';
import GroupMembers from '@/components/GroupMembers';
import GroupChat from '@/components/GroupChat';
import GroupEditModal from '@/components/GroupEditModal';
import styles from '@/styles/GroupPage.module.css';

export default function GroupPage() {
  const { user } = useAuth();
  const { showAlert, showConfirm } = useAlert();
  const { id } = useParams();
  const router = useRouter();
  const [group, setGroup] = useState(null);
  const [isLoading, setIsLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('posts');
  const [isJoining, setIsJoining] = useState(false);
  const [isLeaving, setIsLeaving] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);

  useEffect(() => {
    if (id) {
      fetchGroup();
    }
  }, [id]);

  const fetchGroup = async () => {
    try {
      setIsLoading(true);
      const response = await groupAPI.getGroup(id);

      if (response.data.success) {
        // Handle both possible response structures
        const groupData = response.data.data?.group || response.data.group;
        if (groupData) {
          setGroup(groupData);
        } else {
          throw new Error('Group data not found in response');
        }
      } else {
        throw new Error(response.data.message || 'Failed to fetch group');
      }
    } catch (error) {
      console.error('Error fetching group:', error);
      if (error.response?.status === 404) {
        router.push('/groups');
      } else {
        showAlert({
          type: 'error',
          title: 'Error',
          message: 'Failed to load group. Please try again.'
        });
      }
    } finally {
      setIsLoading(false);
    }
  };

  const handleJoinGroup = async () => {
    if (!group) return;

    setIsJoining(true);
    try {
      const response = await groupAPI.joinGroup(group.id);

      if (response.data.success) {
        // Update group state with request status
        setGroup(prev => ({
          ...prev,
          requestStatus: 'pending'
        }));

        showAlert({
          type: 'info',
          title: 'Request Sent',
          message: 'Join request sent! You will be notified when a group admin responds.'
        });
      }
    } catch (error) {
      console.error('Error joining group:', error);
      showAlert({
        type: 'error',
        title: 'Error',
        message: error.response?.data?.message || 'Failed to send join request. Please try again.'
      });
    } finally {
      setIsJoining(false);
    }
  };

  const handleLeaveGroup = async () => {
    if (!group) return;

    const confirmed = await showConfirm({
      title: 'Leave Group',
      message: 'Are you sure you want to leave this group?',
      confirmText: 'Leave',
      cancelText: 'Cancel',
      confirmVariant: 'danger'
    });

    if (!confirmed) {
      return;
    }

    setIsLeaving(true);
    try {
      const response = await groupAPI.leaveGroup(group.id);

      if (response.data.success) {
        // Update group state
        setGroup(prev => ({
          ...prev,
          isJoined: false,
          membersCount: prev.membersCount - 1
        }));
        showAlert({
          type: 'success',
          title: 'Success',
          message: 'Successfully left the group.'
        });
      }
    } catch (error) {
      console.error('Error leaving group:', error);
      showAlert({
        type: 'error',
        title: 'Error',
        message: error.response?.data?.message || 'Failed to leave group. Please try again.'
      });
    } finally {
      setIsLeaving(false);
    }
  };

  const isGroupAdmin = () => {
    return group && group.isAdmin;
  };

  const isGroupMember = () => {
    return group && user && (group.isJoined || group.creatorId === user.id);
  };

  const canViewContent = () => {
    return group && (group.privacy === 'public' || isGroupMember());
  };

  const handleManageGroup = () => {
    setActiveTab('members');
  };

  const refreshGroupData = async () => {
    await fetchGroup();
  };

  const handleGroupUpdate = async () => {
    await fetchGroup();
  };

  if (isLoading) {
    return (
      <ProtectedRoute>
        <div className={styles.loading}>Loading group...</div>
      </ProtectedRoute>
    );
  }

  if (!group) {
    return (
      <ProtectedRoute>
        <div className={styles.error}>Group not found</div>
      </ProtectedRoute>
    );
  }

  return (
    <ProtectedRoute>
      <div className={styles.groupPageContainer}>
        {/* Group Header */}
        <div className={styles.groupHeader}>
          <div className={styles.coverPhoto}>
            {group.coverPhoto ? (
              <Image
                src={getImageUrl(group.coverPhoto)}
                alt={group.name}
                fill
                sizes="(max-width: 768px) 100vw, (max-width: 1200px) 80vw, 60vw"
                priority={true} // Group cover is above the fold
                style={{ objectFit: 'cover' }}
                onError={(e) => {
                  console.error('Cover photo failed to load:', e.target.src);
                }}
              />
            ) : (
              <div className={styles.defaultCover} />
            )}
          </div>

          <div className={styles.groupInfo}>
            <div className={styles.groupDetails}>
              <h1 className={styles.groupName}>{group.name}</h1>
              <div className={styles.groupMeta}>
                <span className={styles.privacy}>
                  {group.privacy === 'public' ? '🌎 Public Group' : '🔒 Private Group'}
                </span>
                <span className={styles.memberCount}>
                  {group.membersCount} {group.membersCount === 1 ? 'member' : 'members'}
                </span>
              </div>
              {group.description && (
                <p className={styles.groupDescription}>{group.description}</p>
              )}
            </div>

            <div className={styles.groupActions}>
              {isGroupAdmin() ? (
                <>
                  <Button variant="primary" onClick={() => setShowEditModal(true)}>
                    Edit Group
                  </Button>
                  <Button variant="secondary" onClick={handleManageGroup}>
                    Manage Group
                  </Button>
                </>
              ) : isGroupMember() ? (
                <Button
                  variant="outline"
                  onClick={handleLeaveGroup}
                  disabled={isLeaving}
                >
                  {isLeaving ? 'Leaving...' : 'Leave Group'}
                </Button>
              ) : (
                <Button
                  variant={group.requestStatus === 'pending' ? 'outline' : 'primary'}
                  onClick={handleJoinGroup}
                  disabled={isJoining || group.requestStatus === 'pending'}
                >
                  {isJoining ? 'Sending Request...' :
                   group.requestStatus === 'pending' ? 'Request Sent' :
                   group.requestStatus === 'rejected' ? 'Request Again' : 'Request to Join'}
                </Button>
              )}
            </div>
          </div>
        </div>

        {/* Content Area */}
        {canViewContent() ? (
          <>
            {/* Navigation Tabs */}
            <div className={styles.tabNavigation}>
              <button
                className={`${styles.tabButton} ${activeTab === 'posts' ? styles.activeTab : ''}`}
                onClick={() => setActiveTab('posts')}
              >
                Posts
              </button>
              <button
                className={`${styles.tabButton} ${activeTab === 'events' ? styles.activeTab : ''}`}
                onClick={() => setActiveTab('events')}
              >
                Events
              </button>
              <button
                className={`${styles.tabButton} ${activeTab === 'members' ? styles.activeTab : ''}`}
                onClick={() => setActiveTab('members')}
              >
                Members
              </button>
              {isGroupMember() && (
                <button
                  className={`${styles.tabButton} ${activeTab === 'chat' ? styles.activeTab : ''}`}
                  onClick={() => setActiveTab('chat')}
                >
                  Chat
                </button>
              )}
            </div>

            {/* Tab Content */}
            <div className={styles.tabContent}>
              {activeTab === 'posts' && (
                <GroupPosts 
                  groupId={group.id} 
                  isGroupMember={isGroupMember()} 
                  isGroupAdmin={isGroupAdmin()} 
                  groupCreatorId={group.creatorId}
                  refreshGroupData={refreshGroupData} 
                />
              )}
              {activeTab === 'events' && (
                <GroupEvents
                  groupId={group.id}
                  isGroupMember={isGroupMember()}
                  isGroupAdmin={isGroupAdmin()}
                />
              )}
              {activeTab === 'members' && (
                <GroupMembers
                  groupId={group.id}
                  isGroupAdmin={isGroupAdmin()}
                  isGroupMember={isGroupMember()}
                  onMembershipChange={refreshGroupData}
                />
              )}
              {activeTab === 'chat' && (
                <GroupChat
                  groupId={group.id}
                  isVisible={activeTab === 'chat'}
                />
              )}
            </div>
          </>
        ) : (
          <div className={styles.restrictedContent}>
            <div className={styles.restrictedMessage}>
              <h3>This is a private group</h3>
              <p>You need to be a member to see posts and other content.</p>
              <Button
                variant={group.requestStatus === 'pending' ? 'outline' : 'primary'}
                onClick={handleJoinGroup}
                disabled={isJoining || group.requestStatus === 'pending'}
              >
                {isJoining ? 'Sending Request...' :
                 group.requestStatus === 'pending' ? 'Request Sent' :
                 group.requestStatus === 'rejected' ? 'Request Again' : 'Request to Join'}
              </Button>
            </div>
          </div>
        )}

        {/* Group Edit Modal */}
        <GroupEditModal
          group={group}
          isOpen={showEditModal}
          onClose={() => setShowEditModal(false)}
          onUpdate={handleGroupUpdate}
        />
      </div>
    </ProtectedRoute>
  );
}
