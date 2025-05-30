import axios from 'axios';
import { getToken, logout } from './auth';

// Create axios instance with default config
const api = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api',
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add request interceptor to add auth token to requests
api.interceptors.request.use(
  (config) => {
    const token = getToken();
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Add response interceptor to handle auth errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    // Handle 401 Unauthorized errors by logging out
    if (error.response && error.response.status === 401) {
      logout();
      // Redirect to login page if we're in the browser
      if (typeof window !== 'undefined') {
        window.location.href = '/auth/login';
      }
    }
    return Promise.reject(error);
  }
);

// Auth API calls
export const authAPI = {
  login: (email, password) => api.post('/auth/login', { email, password }),
  register: (userData) => api.post('/auth/register', userData),
  logout: () => api.post('/auth/logout'),
};

// User API calls
export const userAPI = {
  getProfile: (userId) => api.get(`/users/${userId}`),
  updateProfile: (userData) => {
    console.log('Sending profile update data:', userData);
    return api.put('/users/profile', userData)
      .then(response => {
        console.log('Profile update response:', response);
        return response;
      })
      .catch(error => {
        console.error('Profile update error:', error.response || error);
        throw error;
      });
  },
  uploadAvatar: (formData) => {
    console.log('Sending avatar upload request with formData:', formData);
    return api.post('/users/avatar', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    })
    .then(response => {
      console.log('Avatar upload success:', response);
      return response;
    })
    .catch(error => {
      console.error('Avatar upload error:', error.response || error);
      throw error;
    });
  },
  uploadCoverPhoto: (formData) => {
    console.log('Sending cover photo upload request with formData:', formData);
    return api.post('/users/cover', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    })
    .then(response => {
      console.log('Cover photo upload success:', response);
      return response;
    })
    .catch(error => {
      console.error('Cover photo upload error:', error.response || error);
      throw error;
    });
  },
  getFollowers: (userId) => api.get(`/users/${userId}/followers`),
  getFollowing: (userId) => api.get(`/users/${userId}/following`),
  follow: (userId) => api.post(`/users/${userId}/follow`),
  unfollow: (userId) => api.delete(`/users/${userId}/follow`),
  getFollowRequests: () => api.get('/users/follow-requests'),
  respondToFollowRequest: (requestId, accept) =>
    api.put(`/users/follow-requests/${requestId}`, { accept }),
};

// Post API calls
export const postAPI = {
  getPosts: (userId) => api.get(`/posts/user/${userId}`),
  getFeed: (page = 1, limit = 10) => api.get(`/posts/feed?page=${page}&limit=${limit}`),
  createPost: (postData) => {
    // For FormData, don't set Content-Type header - let browser set it with boundary
    return api.post('/posts', postData, {
      headers: {
        'Content-Type': undefined, // This removes the default application/json header
      },
    });
  },
  updatePost: (postId, postData) => {
    // For FormData, don't set Content-Type header - let browser set it with boundary
    return api.put(`/posts/${postId}`, postData, {
      headers: {
        'Content-Type': undefined, // This removes the default application/json header
      },
    });
  },
  deletePost: (postId) => api.delete(`/posts/${postId}`),
  likePost: (postId) => api.post(`/posts/${postId}/like`),
  unlikePost: (postId) => api.delete(`/posts/${postId}/like`),
  getComments: (postId) => api.get(`/posts/${postId}/comments`),
  addComment: (postId, content) => api.post(`/posts/${postId}/comments`, { content }),
  deleteComment: (postId, commentId) => api.delete(`/posts/${postId}/comments/${commentId}`),
};

// Group API calls
export const groupAPI = {
  getGroups: () => api.get('/groups'),
  getGroup: (groupId) => api.get(`/groups/${groupId}`),
  createGroup: (groupData) => api.post('/groups', groupData),
  updateGroup: (groupId, groupData) => api.put(`/groups/${groupId}`, groupData),
  deleteGroup: (groupId) => api.delete(`/groups/${groupId}`),
  joinGroup: (groupId) => api.post(`/groups/${groupId}/join`),
  leaveGroup: (groupId) => api.delete(`/groups/${groupId}/join`),
  getGroupPosts: (groupId) => api.get(`/groups/${groupId}/posts`),
  createGroupPost: (groupId, postData) => {
    // For FormData, don't set Content-Type header - let browser set it with boundary
    return api.post(`/groups/${groupId}/posts`, postData, {
      headers: {
        'Content-Type': undefined, // This removes the default application/json header
      },
    });
  },
  createGroupEvent: (groupId, eventData) => api.post(`/groups/${groupId}/events`, eventData),
  getGroupEvents: (groupId) => api.get(`/groups/${groupId}/events`),
  respondToEvent: (eventId, response) => api.post(`/groups/events/${eventId}/respond`, { response }),
};

// Notification API calls
export const notificationAPI = {
  getNotifications: () => api.get('/notifications'),
  markAsRead: (notificationId) => api.put(`/notifications/${notificationId}/read`),
  markAllAsRead: () => api.put('/notifications/read-all'),
};

export default api;
