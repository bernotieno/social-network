'use client';

import { useState } from 'react';

export const useModal = () => {
  const [isOpen, setIsOpen] = useState(false);

  const openModal = () => setIsOpen(true);
  const closeModal = () => setIsOpen(false);
  const toggleModal = () => setIsOpen(!isOpen);

  return {
    isOpen,
    openModal,
    closeModal,
    toggleModal,
  };
};

export const useConfirmModal = () => {
  const [isOpen, setIsOpen] = useState(false);
  const [config, setConfig] = useState({
    title: 'Confirm Action',
    message: '',
    confirmText: 'Confirm',
    cancelText: 'Cancel',
    confirmVariant: 'primary',
    onConfirm: () => {},
  });

  const openConfirmModal = (modalConfig) => {
    setConfig({ ...config, ...modalConfig });
    setIsOpen(true);
  };

  const closeConfirmModal = () => {
    setIsOpen(false);
  };

  const handleConfirm = () => {
    config.onConfirm();
    closeConfirmModal();
  };

  return {
    isOpen,
    config,
    openConfirmModal,
    closeConfirmModal,
    handleConfirm,
  };
};

export const useAlert = () => {
  const [isOpen, setIsOpen] = useState(false);
  const [config, setConfig] = useState({
    title: 'Alert',
    message: '',
    type: 'info',
    buttonText: 'OK',
  });

  const showAlert = (alertConfig) => {
    setConfig({ ...config, ...alertConfig });
    setIsOpen(true);
  };

  const closeAlert = () => {
    setIsOpen(false);
  };

  return {
    isOpen,
    config,
    showAlert,
    closeAlert,
  };
};

export default useModal;
