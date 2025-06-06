'use client';

import { useState, useEffect } from 'react';
import Image from 'next/image';
import { useAuth } from '@/hooks/useAuth';
import { groupAPI } from '@/utils/api';
import { useAlert } from '@/contexts/AlertContext';
import Button from '@/components/Button';
import styles from '@/styles/GroupEvents.module.css';

export default function GroupEvents({ groupId, isGroupMember, isGroupAdmin }) {
  const { user } = useAuth();
  const { showSuccess, showError } = useAlert();
  const [events, setEvents] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [isCreating, setIsCreating] = useState(false);
  const [editingEvent, setEditingEvent] = useState(null);
  const [isUpdating, setIsUpdating] = useState(false);
  const [newEvent, setNewEvent] = useState({
    title: '',
    description: '',
    location: '',
    startTime: '',
    endTime: ''
  });

  useEffect(() => {
    fetchEvents();
  }, [groupId]);

  const fetchEvents = async () => {
    try {
      setIsLoading(true);
      const response = await groupAPI.getGroupEvents(groupId);

      if (response.data.success) {
        setEvents(response.data.data.events || []);
      }
    } catch (error) {
      console.error('Error fetching group events:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleInputChange = (e) => {
    const { name, value } = e.target;
    setNewEvent(prev => ({
      ...prev,
      [name]: value
    }));
  };

  const handleCreateEvent = async (e) => {
    e.preventDefault();

    if (!newEvent.title.trim() || !newEvent.startTime || !newEvent.endTime) {
      showError('Please fill in all required fields', 'Validation Error');
      return;
    }

    // Validate that end time is after start time
    const startDate = new Date(newEvent.startTime);
    const endDate = new Date(newEvent.endTime);

    if (endDate <= startDate) {
      showError('End time must be after start time', 'Validation Error');
      return;
    }

    setIsCreating(true);

    try {
      // Convert datetime-local format to RFC3339 format
      const eventData = {
        ...newEvent,
        startTime: new Date(newEvent.startTime).toISOString(),
        endTime: new Date(newEvent.endTime).toISOString()
      };

      const response = await groupAPI.createGroupEvent(groupId, eventData);

      if (response.data.success) {
        const newEventData = response.data.data.event;
        setEvents(prev => [newEventData, ...prev]);

        // Show success message
        showSuccess('Your event has been created successfully!', 'Event Created');

        // Reset form
        setNewEvent({
          title: '',
          description: '',
          location: '',
          startTime: '',
          endTime: ''
        });
        setShowCreateForm(false);
      }
    } catch (error) {
      console.error('Error creating event:', error);
      showError(error.response?.data?.message || 'Failed to create event. Please try again.');
    } finally {
      setIsCreating(false);
    }
  };

  const handleEventResponse = async (eventId, response) => {
    try {
      await groupAPI.respondToEvent(eventId, response);

      // Update the event in the list
      setEvents(prev => prev.map(event => {
        if (event.id === eventId) {
          const updatedEvent = { ...event };

          // Update counts based on previous and new response
          if (event.userResponse) {
            // Remove from previous response count
            if (event.userResponse === 'going') updatedEvent.goingCount--;
            else if (event.userResponse === 'maybe') updatedEvent.maybeCount--;
            else if (event.userResponse === 'not_going') updatedEvent.declinedCount--;
          }

          // Add to new response count
          if (response === 'going') updatedEvent.goingCount++;
          else if (response === 'maybe') updatedEvent.maybeCount++;
          else if (response === 'not_going') updatedEvent.declinedCount++;

          updatedEvent.userResponse = response;
          return updatedEvent;
        }
        return event;
      }));
    } catch (error) {
      console.error('Error responding to event:', error);
      showError('Failed to update response. Please try again.');
    }
  };

  const handleEditEvent = (event) => {
    setEditingEvent({
      ...event,
      startTime: new Date(event.startTime).toISOString().slice(0, 16),
      endTime: new Date(event.endTime).toISOString().slice(0, 16)
    });
  };

  const handleUpdateEvent = async (e) => {
    e.preventDefault();

    if (!editingEvent.title.trim() || !editingEvent.startTime || !editingEvent.endTime) {
      showError('Please fill in all required fields', 'Validation Error');
      return;
    }

    // Validate that end time is after start time
    const startDate = new Date(editingEvent.startTime);
    const endDate = new Date(editingEvent.endTime);

    if (endDate <= startDate) {
      showError('End time must be after start time', 'Validation Error');
      return;
    }

    setIsUpdating(true);

    try {
      // Convert datetime-local format to RFC3339 format
      const eventData = {
        ...editingEvent,
        startTime: new Date(editingEvent.startTime).toISOString(),
        endTime: new Date(editingEvent.endTime).toISOString()
      };

      const response = await groupAPI.updateGroupEvent(editingEvent.id, eventData);

      if (response.data.success) {
        const updatedEvent = response.data.data.event;
        setEvents(prev => prev.map(event =>
          event.id === updatedEvent.id ? updatedEvent : event
        ));

        showSuccess('Event updated successfully!', 'Event Updated');
        setEditingEvent(null);
      }
    } catch (error) {
      console.error('Error updating event:', error);
      showError(error.response?.data?.message || 'Failed to update event. Please try again.');
    } finally {
      setIsUpdating(false);
    }
  };

  const handleDeleteEvent = async (eventId) => {
    if (!window.confirm('Are you sure you want to delete this event? This action cannot be undone.')) {
      return;
    }

    try {
      await groupAPI.deleteGroupEvent(eventId);
      setEvents(prev => prev.filter(event => event.id !== eventId));
      showSuccess('Event deleted successfully!', 'Event Deleted');
    } catch (error) {
      console.error('Error deleting event:', error);
      showError(error.response?.data?.message || 'Failed to delete event. Please try again.');
    }
  };

  const handleEditInputChange = (e) => {
    const { name, value } = e.target;
    setEditingEvent(prev => ({
      ...prev,
      [name]: value
    }));
  };

  const canEditEvent = (event) => {
    return user && (event.creatorId === user.id || isGroupAdmin);
  };

  const formatDateTime = (dateTimeString) => {
    const date = new Date(dateTimeString);
    return date.toLocaleString('en-US', {
      weekday: 'short',
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  const isEventPast = (endTime) => {
    return new Date(endTime) < new Date();
  };

  if (isLoading) {
    return <div className={styles.loading}>Loading events...</div>;
  }

  return (
    <div className={styles.groupEventsContainer}>
      {/* Create Event Button */}
      {isGroupMember && !showCreateForm && (
        <div className={styles.createEventSection}>
          <Button
            variant="primary"
            onClick={() => setShowCreateForm(true)}
            fullWidth
          >
            Create Event
          </Button>
        </div>
      )}

      {/* Create Event Form */}
      {showCreateForm && (
        <div className={styles.createEventForm}>
          <div className={styles.formHeader}>
            <h3>Create New Event</h3>
            <button
              type="button"
              className={styles.closeButton}
              onClick={() => setShowCreateForm(false)}
            >
              ✕
            </button>
          </div>

          <form onSubmit={handleCreateEvent}>
            <div className={styles.inputGroup}>
              <label htmlFor="title" className={styles.label}>Event Title *</label>
              <input
                type="text"
                id="title"
                name="title"
                value={newEvent.title}
                onChange={handleInputChange}
                placeholder="Enter event title"
                className={styles.input}
                required
              />
            </div>

            <div className={styles.inputGroup}>
              <label htmlFor="description" className={styles.label}>Description</label>
              <textarea
                id="description"
                name="description"
                value={newEvent.description}
                onChange={handleInputChange}
                placeholder="Describe your event..."
                className={styles.textarea}
                rows={3}
              />
            </div>

            <div className={styles.inputGroup}>
              <label htmlFor="location" className={styles.label}>Location</label>
              <input
                type="text"
                id="location"
                name="location"
                value={newEvent.location}
                onChange={handleInputChange}
                placeholder="Event location"
                className={styles.input}
              />
            </div>

            <div className={styles.dateTimeRow}>
              <div className={styles.inputGroup}>
                <label htmlFor="startTime" className={styles.label}>Start Time *</label>
                <input
                  type="datetime-local"
                  id="startTime"
                  name="startTime"
                  value={newEvent.startTime}
                  onChange={handleInputChange}
                  className={styles.input}
                  required
                />
              </div>

              <div className={styles.inputGroup}>
                <label htmlFor="endTime" className={styles.label}>End Time *</label>
                <input
                  type="datetime-local"
                  id="endTime"
                  name="endTime"
                  value={newEvent.endTime}
                  onChange={handleInputChange}
                  className={styles.input}
                  required
                />
              </div>
            </div>

            <div className={styles.formActions}>
              <Button
                type="button"
                variant="secondary"
                onClick={() => setShowCreateForm(false)}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                variant="primary"
                disabled={isCreating}
              >
                {isCreating ? 'Creating...' : 'Create Event'}
              </Button>
            </div>
          </form>
        </div>
      )}

      {/* Events List */}
      <div className={styles.eventsList}>
        {events.length === 0 ? (
          <div className={styles.emptyEvents}>
            <p>No events yet</p>
            {isGroupMember && (
              <p>Create the first event for this group!</p>
            )}
          </div>
        ) : (
          events.map(event => (
            <div key={event.id} className={styles.eventCard}>
              <div className={styles.eventHeader}>
                <div className={styles.eventInfo}>
                  <h3 className={styles.eventTitle}>{event.title}</h3>
                  <div className={styles.eventMeta}>
                    <div className={styles.eventTime}>
                      📅 {formatDateTime(event.startTime)}
                      {event.endTime && ` - ${formatDateTime(event.endTime)}`}
                    </div>
                    {event.location && (
                      <div className={styles.eventLocation}>
                        📍 {event.location}
                      </div>
                    )}
                  </div>
                </div>
                <div className={styles.eventHeaderActions}>
                  {isEventPast(event.endTime) && (
                    <div className={styles.pastEventBadge}>Past Event</div>
                  )}
                  {canEditEvent(event) && !isEventPast(event.endTime) && (
                    <div className={styles.eventManageActions}>
                      <Button
                        variant="secondary"
                        size="small"
                        onClick={() => handleEditEvent(event)}
                      >
                        Edit
                      </Button>
                      <Button
                        variant="danger"
                        size="small"
                        onClick={() => handleDeleteEvent(event.id)}
                      >
                        Delete
                      </Button>
                    </div>
                  )}
                </div>
              </div>

              {editingEvent && editingEvent.id === event.id ? (
                <form onSubmit={handleUpdateEvent} className={styles.editEventForm}>
                  <div className={styles.inputGroup}>
                    <label className={styles.label}>Event Title *</label>
                    <input
                      type="text"
                      name="title"
                      value={editingEvent.title}
                      onChange={handleEditInputChange}
                      className={styles.input}
                      required
                    />
                  </div>

                  <div className={styles.inputGroup}>
                    <label className={styles.label}>Description</label>
                    <textarea
                      name="description"
                      value={editingEvent.description}
                      onChange={handleEditInputChange}
                      className={styles.textarea}
                      rows="3"
                    />
                  </div>

                  <div className={styles.inputGroup}>
                    <label className={styles.label}>Location</label>
                    <input
                      type="text"
                      name="location"
                      value={editingEvent.location}
                      onChange={handleEditInputChange}
                      className={styles.input}
                    />
                  </div>

                  <div className={styles.dateTimeRow}>
                    <div className={styles.inputGroup}>
                      <label className={styles.label}>Start Time *</label>
                      <input
                        type="datetime-local"
                        name="startTime"
                        value={editingEvent.startTime}
                        onChange={handleEditInputChange}
                        className={styles.input}
                        required
                      />
                    </div>

                    <div className={styles.inputGroup}>
                      <label className={styles.label}>End Time *</label>
                      <input
                        type="datetime-local"
                        name="endTime"
                        value={editingEvent.endTime}
                        onChange={handleEditInputChange}
                        className={styles.input}
                        required
                      />
                    </div>
                  </div>

                  <div className={styles.formActions}>
                    <Button
                      type="button"
                      variant="secondary"
                      onClick={() => setEditingEvent(null)}
                      disabled={isUpdating}
                    >
                      Cancel
                    </Button>
                    <Button
                      type="submit"
                      variant="primary"
                      disabled={isUpdating}
                    >
                      {isUpdating ? 'Updating...' : 'Update Event'}
                    </Button>
                  </div>
                </form>
              ) : (
                <>
                  {event.description && (
                    <p className={styles.eventDescription}>{event.description}</p>
                  )}

                  <div className={styles.eventStats}>
                <div className={styles.statItem}>
                  <span className={styles.statNumber}>{event.goingCount || 0}</span>
                  <span className={styles.statLabel}>Going</span>
                </div>
                <div className={styles.statItem}>
                  <span className={styles.statNumber}>{event.declinedCount || 0}</span>
                  <span className={styles.statLabel}>Can't Go</span>
                </div>
                  </div>

                  {isGroupMember && !isEventPast(event.endTime) && (
                    <div className={styles.eventActions}>
                      <Button
                        variant={event.userResponse === 'going' ? 'primary' : 'outline'}
                        onClick={() => handleEventResponse(event.id, 'going')}
                        size="small"
                      >
                        Going
                      </Button>
                      <Button
                        variant={event.userResponse === 'not_going' ? 'primary' : 'outline'}
                        onClick={() => handleEventResponse(event.id, 'not_going')}
                        size="small"
                      >
                        Can&apos;t Go
                      </Button>
                    </div>
                  )}
                </>
              )}
            </div>
          ))
        )}
      </div>
    </div>
  );
}
