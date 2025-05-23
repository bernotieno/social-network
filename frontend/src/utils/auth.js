import Cookies from 'js-cookie';

// Cookie names
const TOKEN_COOKIE = 'social_network_session'; // Session token cookie
const USER_COOKIE = 'social_network_session'; // User data cookie

// Session expiration in days
const SESSION_EXPIRATION = 7;

/**
 * Set authentication token and user data in cookies
 * @param {string} token - Session token
 * @param {object} user - User data
 */
export const setAuth = (token, user) => {
  Cookies.set(TOKEN_COOKIE, token, { expires: SESSION_EXPIRATION });
  Cookies.set(USER_COOKIE, JSON.stringify(user), { expires: SESSION_EXPIRATION });
};

/**
 * Get authentication token from cookies
 * @returns {string|null} - Session token or null if not found
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
  if (!userData) return null;

  try {
    return JSON.parse(userData);
  } catch (error) {
    // If parsing fails, remove the invalid cookie to prevent future errors
    console.error('Error parsing user data from cookie:', error);
    Cookies.remove(USER_COOKIE);
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
      expires: SESSION_EXPIRATION
    });
  }
};
