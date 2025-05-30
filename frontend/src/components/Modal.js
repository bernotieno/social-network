'use client';

import React, { useEffect } from 'react';
import Button from '@/components/Button';
import styles from '@/styles/Modal.module.css';

const Modal = ({ 
  isOpen, 
  onClose, 
  title, 
  children, 
  showCloseButton = true,
  closeOnOverlayClick = true 
}) => {
  // Close modal on Escape key
  useEffect(() => {
    const handleEscape = (e) => {
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };

    if (isOpen) {
      document.addEventListener('keydown', handleEscape);
      document.body.style.overflow = 'hidden'; // Prevent background scrolling
    }

    return () => {
      document.removeEventListener('keydown', handleEscape);
      document.body.style.overflow = 'unset';
    };
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const handleOverlayClick = (e) => {
    if (e.target === e.currentTarget && closeOnOverlayClick) {
      onClose();
    }
  };

  return (
    <div className={styles.overlay} onClick={handleOverlayClick}>
      <div className={styles.modal}>
        {(title || showCloseButton) && (
          <div className={styles.header}>
            {title && <h2 className={styles.title}>{title}</h2>}
            {showCloseButton && (
              <button 
                className={styles.closeButton}
                onClick={onClose}
                aria-label="Close modal"
              >
                ×
              </button>
            )}
          </div>
        )}
        
        <div className={styles.content}>
          {children}
        </div>
      </div>
    </div>
  );
};

// Confirmation Modal Component
export const ConfirmModal = ({ 
  isOpen, 
  onClose, 
  onConfirm, 
  title = "Confirm Action",
  message,
  confirmText = "Confirm",
  cancelText = "Cancel",
  confirmVariant = "danger",
  isLoading = false
}) => {
  const handleConfirm = () => {
    onConfirm();
  };

  return (
    <Modal 
      isOpen={isOpen} 
      onClose={onClose} 
      title={title}
      closeOnOverlayClick={!isLoading}
    >
      <div className={styles.confirmContent}>
        <p className={styles.message}>{message}</p>
        
        <div className={styles.actions}>
          <Button
            variant="secondary"
            onClick={onClose}
            disabled={isLoading}
          >
            {cancelText}
          </Button>
          <Button
            variant={confirmVariant}
            onClick={handleConfirm}
            disabled={isLoading}
          >
            {isLoading ? 'Processing...' : confirmText}
          </Button>
        </div>
      </div>
    </Modal>
  );
};

// Alert Modal Component
export const AlertModal = ({ 
  isOpen, 
  onClose, 
  title = "Alert",
  message,
  type = "info", // info, success, warning, error
  buttonText = "OK"
}) => {
  return (
    <Modal 
      isOpen={isOpen} 
      onClose={onClose} 
      title={title}
    >
      <div className={styles.alertContent}>
        <div className={`${styles.alertIcon} ${styles[type]}`}>
          {type === 'success' && '✓'}
          {type === 'error' && '✕'}
          {type === 'warning' && '⚠'}
          {type === 'info' && 'ℹ'}
        </div>
        
        <p className={styles.message}>{message}</p>
        
        <div className={styles.actions}>
          <Button
            variant="primary"
            onClick={onClose}
          >
            {buttonText}
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default Modal;
