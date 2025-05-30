'use client';

import { useState } from 'react';
import Link from 'next/link';
import Image from 'next/image';
import { useAuth } from '@/hooks/useAuth';
import useNotifications from '@/hooks/useNotifications';
import { getImageUrl } from '@/utils/images';
import styles from '@/styles/Navbar.module.css';

const Navbar = () => {
  const { user, isAuthenticated, logout } = useAuth();
  const { unreadCount } = useNotifications();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const toggleMobileMenu = () => {
    setMobileMenuOpen(!mobileMenuOpen);
  };

  return (
    <nav className={styles.navbar}>
      <div className={styles.container}>
        <div className={styles.logo}>
          <Link href="/">
            <span className={styles.logoText}>SocialNetwork</span>
          </Link>
        </div>

        {/* Mobile menu button */}
        <button
          className={styles.mobileMenuButton}
          onClick={toggleMobileMenu}
          aria-label="Toggle menu"
        >
          <span className={styles.hamburger}></span>
        </button>

        {/* Navigation links */}
        <div className={`${styles.navLinks} ${mobileMenuOpen ? styles.active : ''}`}>
          {isAuthenticated ? (
            <>
              <Link href="/" className={styles.navLink}>
                Home
              </Link>
              <Link href="/posts/create" className={styles.navLink}>
                Create Post
              </Link>
              <Link href="/groups" className={styles.navLink}>
                Groups
              </Link>
              <Link href="/chat" className={styles.navLink}>
                Chat
              </Link>
              <Link href="/notifications" className={styles.navLink}>
                Notifications
                {unreadCount > 0 && (
                  <span className={styles.badge}>{unreadCount}</span>
                )}
              </Link>
              <div className={styles.profileDropdown}>
                <button className={styles.profileButton}>
                  {user?.profilePicture ? (
                    <Image
                      src={getImageUrl(user.profilePicture)}
                      alt={user.username}
                      width={32}
                      height={32}
                      className={styles.avatar}
                    />
                  ) : (
                    <div className={styles.avatarPlaceholder}>
                      {user?.username?.charAt(0).toUpperCase() || 'U'}
                    </div>
                  )}
                </button>
                <div className={styles.dropdownContent}>
                  <Link href={`/profile/${user?.id}`} className={styles.dropdownItem}>
                    Profile
                  </Link>
                  <button onClick={logout} className={styles.dropdownItem}>
                    Logout
                  </button>
                </div>
              </div>
            </>
          ) : (
            <>
              <Link href="/auth/login" className={styles.navLink}>
                Login
              </Link>
              <Link href="/auth/register" className={styles.navLink}>
                Register
              </Link>
            </>
          )}
        </div>
      </div>
    </nav>
  );
};

export default Navbar;
