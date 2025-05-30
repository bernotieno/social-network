/**
 * Utility functions for handling images in the application
 */

/**
 * Resolves an image path to a full URL
 * @param {string} imagePath - The image path from the API (e.g., "/uploads/avatars/...")
 * @returns {string|null} - Full URL to the image or null if no path provided
 */
export const getImageUrl = (imagePath) => {
  if (!imagePath) return null;
  if (imagePath.startsWith('http')) return imagePath; // Already a full URL

  // Get the backend API URL and remove /api suffix since images are served directly
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';
  const backendUrl = apiUrl.replace('/api', '');

  return `${backendUrl}${imagePath}`;
};

/**
 * Gets a user's profile picture URL
 * @param {object} user - User object with profilePicture property
 * @returns {string|null} - Full URL to profile picture or null
 */
export const getUserProfilePictureUrl = (user) => {
  return getImageUrl(user?.profilePicture);
};

/**
 * Gets a user's cover photo URL
 * @param {object} user - User object with coverPhoto property
 * @returns {string|null} - Full URL to cover photo or null
 */
export const getUserCoverPhotoUrl = (user) => {
  return getImageUrl(user?.coverPhoto);
};

/**
 * Gets a fallback avatar for users without profile pictures
 * @param {object} user - User object with username property
 * @returns {string} - Data URL for a generated avatar
 */
export const getFallbackAvatar = (user) => {
  const initial = user?.username?.charAt(0).toUpperCase() || user?.fullName?.charAt(0).toUpperCase() || 'U';

  // Create a simple SVG avatar with the user's initial
  const svg = `
    <svg width="120" height="120" xmlns="http://www.w3.org/2000/svg">
      <circle cx="60" cy="60" r="60" fill="#1877f2"/>
      <text x="60" y="75" font-family="Arial, sans-serif" font-size="48" font-weight="bold"
            text-anchor="middle" fill="white">${initial}</text>
    </svg>
  `;

  return `data:image/svg+xml;base64,${btoa(svg)}`;
};
