import Cookies from 'js-cookie';

// Cookie names
const TOKEN_COOKIE = 'social_network_token';
const USER_COOKIE = 'social_network_user';

// Token expiration in days
const TOKEN_EXPIRATION = 7;

/**
 * Set authentication token and user data in cookies
 * @param {string} token - JWT token
 * @param {object} user - User data
 */
export const setAuth = (token, user) => {
  Cookies.set(TOKEN_COOKIE, token, { expires: TOKEN_EXPIRATION });
  Cookies.set(USER_COOKIE, JSON.stringify(user), { expires: TOKEN_EXPIRATION });
};

/**
 * Get authentication token from cookies
 * @returns {string|null} - JWT token or null if not found
 */
export const getToken = () => {
  return Cookies.get(TOKEN_COOKIE) || null;
};

/**
 * Get user data from cookies
 * @returns {object|null} - User data or null if not found
 */
export const getUser = () => {
  const userData = Cookies.get(USER_COOKIE);
  return userData;
};

/**
 * Check if user is authenticated
 * @returns {boolean} - True if user is authenticated
 */
export const isAuthenticated = () => {
  return !!getToken();
};

/**
 * Remove authentication token and user data from cookies
 */
export const logout = () => {
  Cookies.remove(TOKEN_COOKIE);
  Cookies.remove(USER_COOKIE);
};

/**
 * Update user data in cookies
 * @param {object} userData - Updated user data
 */
export const updateUserData = (userData) => {
  const currentUser = getUser();
  if (currentUser) {
    Cookies.set(USER_COOKIE, JSON.stringify({ ...currentUser, ...userData }), { 
      expires: TOKEN_EXPIRATION 
    });
  }
};
