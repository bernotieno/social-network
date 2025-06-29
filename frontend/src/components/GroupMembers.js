'use client';

import { useState, useEffect } from 'react';
import Image from 'next/image';
import Link from 'next/link';
import { useAuth } from '@/hooks/useAuth';

import { groupAPI, userAPI } from '@/utils/api';
import { getUserProfilePictureUrl, getFallbackAvatar } from '@/utils/images';
import { useAlert } from '@/contexts/AlertContext';
import Button from '@/components/Button';
import styles from '@/styles/GroupMembers.module.css';

export default function GroupMembers({ groupId, isGroupAdmin, isGroupMember, onMembershipChange }) {
  const { user } = useAuth();
  const { showConfirm,showAlert,showError, showSuccess } = useAlert();
  const [members, setMembers] = useState([]);
  const [pendingRequests, setPendingRequests] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('members');
  const [showInviteModal, setShowInviteModal] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState([]);
  const [isSearching, setIsSearching] = useState(false);

  useEffect(() => {
    if (isGroupMember) {
      fetchMembers();
        fetchPendingRequests();
    }
  }, [groupId, isGroupMember, isGroupAdmin]);

  const fetchMembers = async () => {
    try {
      setIsLoading(true);
      const response = await groupAPI.getGroupMembers(groupId);

      if (response.data.success) {
        setMembers(response.data.data.members || []);
      }
    } catch (error) {
      console.error('Error fetching group members:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const fetchPendingRequests = async () => {
    try {
      const response = await groupAPI.getGroupPendingRequests(groupId);

      if (response.data.success) {
        setPendingRequests(response.data.data.requests || []);
      }
    } catch (error) {
      console.error('Error fetching pending requests:', error);
    }
  };

  const handleApproveRequest = async (userId) => {
    try {
      await groupAPI.approveJoinRequest(groupId, userId);

      // Move from pending to members
      const approvedRequest = pendingRequests.find(req => req.userId === userId);
      if (approvedRequest) {
        setPendingRequests(prev => prev.filter(req => req.userId !== userId));
        setMembers(prev => [...prev, { ...approvedRequest, role: 'member', status: 'accepted' }]);
      }
    } catch (error) {
      console.error('Error approving request:', error);
      showError('Failed to approve request. Please try again.');
    }
  };

  const handleRejectRequest = async (userId) => {
    try {
      await groupAPI.rejectJoinRequest(groupId, userId);

      // Remove from pending requests
      setPendingRequests(prev => prev.filter(req => req.userId !== userId));
    } catch (error) {
      console.error('Error rejecting request:', error);
      showError('Failed to reject request. Please try again.');
    }
  };

  const searchUsers = async (query) => {
    if (!query.trim()) {
      setSearchResults([]);
      return;
    }

    setIsSearching(true);
    try {
      const response = await userAPI.searchUsers(query);

      if (response.data && response.data.success) {
        // Get users array from response - handle both possible structures
        const users = response.data.data?.users || response.data.users || [];

        if (!Array.isArray(users)) {
          console.error('Users data is not an array:', users);
          setSearchResults([]);
          return;
        }

        // Filter out users who are already members
        const memberIds = members.map(member => {
          // Handle different possible member structures
          return member.userId || member.user?.id || member.id;
        });

        const filteredResults = users.filter(
          user => !memberIds.includes(user.id)
        );

        setSearchResults(filteredResults);
      } else {
        console.error('API response not successful:', response.data);
        setSearchResults([]);
      }
    } catch (error) {
      console.error('Error searching users:', error);
      setSearchResults([]);
    } finally {
      setIsSearching(false);
    }
  };

  const handleInviteUser = async (userId) => {
    try {
      await groupAPI.inviteToGroup(groupId, userId);

      // Remove from search results
      setSearchResults(prev => prev.filter(user => user.id !== userId));

      // Note: We don't refresh members list since invited users are not members yet
      // They will only become members after accepting the invitation

      showSuccess('Invitation sent successfully! The user will receive a notification.');
    } catch (error) {
      console.error('Error inviting user:', error);
      showError(error.response?.data?.message || 'Failed to send invitation. Please try again.');
    }
  };

  const handlePromoteMember = async (userId, userName) => {
    // if (!window.confirm(`Are you sure you want to promote ${userName} to admin?`)) {
    //   return;
    // }
    const confirmed = await showConfirm({
      title: 'Promote Member',
      message: `Are you sure you want to promote ${userName} to admin?`,
      confirmText: 'Promote',
      cancelText: 'Cancel',
      confirmVariant: 'primary'  // Using primary color for non-destructive action
    });
  
    if (!confirmed) {
      return;
    }

    try {
      await groupAPI.promoteGroupMember(groupId, userId);
      setMembers(prev => prev.map(member =>
        (member.userId || member.user?.id || member.id) === userId
          ? { ...member, role: 'admin' }
          : member
      ));
    } catch (error) {
      console.error('Error promoting member:', error);
      showError(error.response?.data?.message || 'Failed to promote member. Please try again.');
    }
  };

  const handleDemoteMember = async (userId, userName) => {
    const confirmed = await showConfirm({
      title: 'Promote Member',
      message: `Are you sure you want to demote ${userName} from admin?`,
      confirmText: 'Demote',
      cancelText: 'Cancel',
      confirmVariant: 'primary'  // Using primary color for non-destructive action
    });
    if (!confirmed) {
      return;
    }

    try {
      await groupAPI.demoteGroupMember(groupId, userId);
      setMembers(prev => prev.map(member =>
        (member.userId || member.user?.id || member.id) === userId
          ? { ...member, role: 'member' }
          : member
      ));
      showSuccess(`${userName} demoted from admin successfully!`);
    } catch (error) {
      console.error('Error demoting member:', error);
      showError(error.response?.data?.message || 'Failed to demote member. Please try again.');
    }
  };

  const handleRemoveMember = async (userId, userName) => {
    const confirmed = await showConfirm({
      title: 'Remove Member',
      message: `Are you sure you want to remove ${userName} from this group? This action cannot be undone.`,
      confirmText: 'Remove',
      cancelText: 'Cancel',
      confirmVariant: 'primary'  // Using primary color for non-destructive action
    });
    if (!confirmed)  {
      return;
    }

    try {
      await groupAPI.removeGroupMember(groupId, userId);

      // Remove from members list
      setMembers(prev => prev.filter(member => {
        const memberUserId = member.userId || member.user?.id || member.id;
        return memberUserId !== userId;
      }));

      showSuccess('Member removed successfully!');

      // Notify parent component about membership change
      if (onMembershipChange) {
        onMembershipChange();
      }
    } catch (error) {
      console.error('Error removing member:', error);
      showError(error.response?.data?.message || 'Failed to remove member. Please try again.');
    }
  };

  useEffect(() => {
    const timeoutId = setTimeout(() => {
      searchUsers(searchQuery);
    }, 300);

    return () => clearTimeout(timeoutId);
  }, [searchQuery]);

  if (!isGroupMember) {
    return (
      <div className={styles.restrictedAccess}>
        <p>You need to be a member to view group members.</p>
      </div>
    );
  }

  if (isLoading) {
    return <div className={styles.loading}>Loading members...</div>;
  }

  return (
    <div className={styles.groupMembersContainer}>
      {/* Tab Navigation */}
      <div className={styles.tabNavigation}>
        <button
          className={`${styles.tabButton} ${activeTab === 'members' ? styles.activeTab : ''}`}
          onClick={() => setActiveTab('members')}
        >
          Members ({members.length})
        </button>
        <button
          className={`${styles.tabButton} ${activeTab === 'requests' ? styles.activeTab : ''}`}
          onClick={() => setActiveTab('requests')}
        >
          Requests ({pendingRequests.length})
        </button>
      </div>

      {/* Invite Button */}
      {activeTab === 'members' && (
        <div className={styles.inviteSection}>
          <Button
            variant="primary"
            onClick={() => setShowInviteModal(true)}
          >
            Invite Members
          </Button>
        </div>
      )}

      {/* Members List */}
      {activeTab === 'members' && (
        <div className={styles.membersList}>
          {members.length === 0 ? (
            <div className={styles.emptyState}>
              <p>No members yet</p>
            </div>
          ) : (
            members.map(member => {
              const memberUser = member.user || member;
              return (
                <div key={member.id || member.userId} className={styles.memberCard}>
                  <Link href={`/profile/${memberUser.id}`} className={styles.memberLink}>
                    <div className={styles.memberAvatar}>
                      {memberUser.profilePicture ? (
                        <Image
                          src={getUserProfilePictureUrl(memberUser)}
                          alt={memberUser.username}
                          width={50}
                          height={50}
                          className={styles.avatar}
                          onError={(e) => {
                            e.target.src = getFallbackAvatar(memberUser);
                          }}
                        />
                      ) : (
                        <Image
                          src={getFallbackAvatar(memberUser)}
                          alt={memberUser.username}
                          width={50}
                          height={50}
                          className={styles.avatar}
                        />
                      )}
                    </div>
                    <div className={styles.memberInfo}>
                      <div className={styles.memberNameRow}>
                        <h3 className={styles.memberName}>{memberUser.fullName}</h3>
                        <span className={`${styles.privacyTag} ${memberUser.isPrivate ? styles.privateTag : styles.publicTag}`}>
                          {memberUser.isPrivate ? '🔒' : '🌐'}
                        </span>
                      </div>
                      <p className={styles.memberUsername}>@{memberUser.username}</p>
                    </div>
                  </Link>
                  <div className={styles.memberRole}>
                    {member.role === 'admin' && (
                      <span className={styles.adminBadge}>Admin</span>
                    )}
                    {member.role === 'creator' && (
                      <span className={styles.adminBadge}>Owner</span>
                    )}
                    {isGroupAdmin && memberUser.id !== user?.id && (
                      <div className={styles.memberActions}>
                        {member.role === 'member' ? (
                          <Button
                            variant="primary"
                            size="small"
                            onClick={() => handlePromoteMember(memberUser.id, memberUser.fullName)}
                          >
                            Promote to Admin
                          </Button>
                        ) : (
                          <Button
                            variant="outline"
                            size="small"
                            onClick={() => handleDemoteMember(memberUser.id, memberUser.fullName)}
                          >
                            Demote from Admin
                          </Button>
                        )}
                        <Button
                          variant="danger"
                          size="small"
                          onClick={() => handleRemoveMember(memberUser.id, memberUser.fullName)}
                        >
                          Remove
                        </Button>
                      </div>
                    )}
                  </div>
                </div>
              );
            })
          )}
        </div>
      )}

      {/* Pending Requests */}
      {activeTab === 'requests' && (
        <div className={styles.requestsList}>
          {pendingRequests.length === 0 ? (
            <div className={styles.emptyState}>
              <p>No pending requests</p>
            </div>
          ) : (
            pendingRequests.map(request => {
              const requestUser = request.user || request;
              return (
                <div key={request.id || request.userId} className={styles.requestCard}>
                  <Link href={`/profile/${requestUser.id}`} className={styles.requestLink}>
                    <div className={styles.requestAvatar}>
                      {requestUser.profilePicture ? (
                        <Image
                          src={getUserProfilePictureUrl(requestUser)}
                          alt={requestUser.username}
                          width={50}
                          height={50}
                          className={styles.avatar}
                          onError={(e) => {
                            e.target.src = getFallbackAvatar(requestUser);
                          }}
                        />
                      ) : (
                        <Image
                          src={getFallbackAvatar(requestUser)}
                          alt={requestUser.username}
                          width={50}
                          height={50}
                          className={styles.avatar}
                        />
                      )}
                    </div>
                    <div className={styles.requestInfo}>
                      <div className={styles.requestNameRow}>
                        <h3 className={styles.requestName}>{requestUser.fullName}</h3>
                        <span className={`${styles.privacyTag} ${requestUser.isPrivate ? styles.privateTag : styles.publicTag}`}>
                          {requestUser.isPrivate ? '🔒' : '🌐'}
                        </span>
                      </div>
                      <p className={styles.requestUsername}>@{requestUser.username}</p>
                    </div>
                  </Link>
                  <div className={styles.requestActions}>
                    <Button
                      variant="primary"
                      size="small"
                      onClick={() => handleApproveRequest(requestUser.id)}
                    >
                      Approve
                    </Button>
                    <Button
                      variant="outline"
                      size="small"
                      onClick={() => handleRejectRequest(requestUser.id)}
                    >
                      Reject
                    </Button>
                  </div>
                </div>
              );
            })
          )}
        </div>
      )}

      {/* Invite Modal */}
      {showInviteModal && (
        <div className={styles.modalOverlay} onClick={() => setShowInviteModal(false)}>
          <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
            <div className={styles.modalHeader}>
              <h3>Invite Members</h3>
              <button
                className={styles.closeButton}
                onClick={() => setShowInviteModal(false)}
              >
                ✕
              </button>
            </div>

            <div className={styles.searchSection}>
              <input
                type="text"
                placeholder="Search users to invite..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className={styles.searchInput}
              />
            </div>

            <div className={styles.searchResults}>
              {isSearching ? (
                <div className={styles.searchLoading}>Searching...</div>
              ) : searchResults.length === 0 ? (
                searchQuery.trim() && (
                  <div className={styles.noResults}>No users found</div>
                )
              ) : (
                searchResults.map(user => (
                  <div key={user.id} className={styles.searchResultItem}>
                    <div className={styles.userInfo}>
                      <div className={styles.userAvatar}>
                        {user.profilePicture ? (
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
                      <div>
                        <div className={styles.userNameRow}>
                          <h4 className={styles.userName}>{user.fullName}</h4>
                          <span className={`${styles.privacyTag} ${user.isPrivate ? styles.privateTag : styles.publicTag}`}>
                            {user.isPrivate ? '🔒' : '🌐'}
                          </span>
                        </div>
                        <p className={styles.userUsername}>@{user.username}</p>
                      </div>
                    </div>
                    <Button
                      variant="primary"
                      size="small"
                      onClick={() => handleInviteUser(user.id)}
                    >
                      Invite
                    </Button>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
