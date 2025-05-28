// Local storage keys
const TOKEN_KEY = 'social_network_token';
const USER_KEY = 'social_network_user';

/**
 * Set authentication token and user data in localStorage
 * @param {string} token - Session token
 * @param {object} user - User data
 */
export const setAuth = (token, user) => {
  localStorage.setItem(TOKEN_KEY, token);
  localStorage.setItem(USER_KEY, JSON.stringify(user));
};

/**
 * Get authentication token from localStorage
 * @returns {string|null} - Session token or null if not found
 */
export const getToken = () => {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem(TOKEN_KEY) || null;
};

/**
 * Get user data from localStorage
 * @returns {object|null} - User data or null if not found
 */
export const getUser = () => {
  if (typeof window === 'undefined') return null;

  const userData = localStorage.getItem(USER_KEY);
  if (!userData) return null;

  try {
    return JSON.parse(userData);
  } catch (error) {
    // If parsing fails, remove the invalid data to prevent future errors
    console.error('Error parsing user data from localStorage:', error);
    localStorage.removeItem(USER_KEY);
    return null;
  }
};

/**
 * Check if user is authenticated
 * @returns {boolean} - True if user is authenticated
 */
export const isAuthenticated = () => {
  return !!getToken();
};

/**
 * Remove authentication token and user data from localStorage
 */
export const logout = () => {
  if (typeof window === 'undefined') return;
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
};

/**
 * Update user data in localStorage
 * @param {object} userData - Updated user data
 */
export const updateUserData = (userData) => {
  if (typeof window === 'undefined') return;

  const currentUser = getUser();
  if (currentUser) {
    localStorage.setItem(USER_KEY, JSON.stringify({ ...currentUser, ...userData }));
  }
};
