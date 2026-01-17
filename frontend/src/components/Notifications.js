import React, { useState, useEffect } from 'react';
import api from '../utils/api';

const Notifications = () => {
    const [notifications, setNotifications] = useState([]);
    const [loading, setLoading] = useState(false);
    const [unreadCount, setUnreadCount] = useState(0);

    useEffect(() => {
        fetchNotifications();
        const interval = setInterval(fetchNotifications, 5000); // Обновляем каждые 5 сек
        return () => clearInterval(interval);
    }, []);

    const fetchNotifications = async () => {
        try {
            setLoading(true);
            const response = await api.get('/api/dashboard/notifications');
            const notifs = response.data || [];
            setNotifications(notifs);
            setUnreadCount(notifs.filter(n => !n.is_read).length);
        } catch (error) {
            console.error('Error fetching notifications:', error);
        } finally {
            setLoading(false);
        }
    };

    const handleMarkAsRead = async (notificationId) => {
        try {
            await api.put(`/api/dashboard/notifications/${notificationId}/read`);
            setNotifications(prev => 
                prev.map(n => n.id === notificationId ? { ...n, is_read: true } : n)
            );
            setUnreadCount(prev => Math.max(0, prev - 1));
        } catch (error) {
            console.error('Error marking notification as read:', error);
        }
    };

    const getNotificationIcon = (notificationType) => {
        switch (notificationType) {
            case 'project_assigned':
                return '📋';
            case 'task_assigned':
                return '✅';
            case 'comment':
                return '💬';
            case 'reminder':
                return '🔔';
            default:
                return '📢';
        }
    };

    const formatDate = (dateString) => {
        const date = new Date(dateString);
        const now = new Date();
        const diffMs = now - date;
        const diffMins = Math.floor(diffMs / 60000);
        const diffHours = Math.floor(diffMs / 3600000);
        const diffDays = Math.floor(diffMs / 86400000);

        if (diffMins < 1) return 'Только что';
        if (diffMins < 60) return `${diffMins} мин назад`;
        if (diffHours < 24) return `${diffHours} ч назад`;
        if (diffDays < 7) return `${diffDays} д назад`;
        
        return date.toLocaleDateString('ru-RU');
    };

    return (
        <div className="notifications-panel">
            <div className="notifications-header">
                <h3>Уведомления</h3>
                {unreadCount > 0 && (
                    <span className="unread-badge">{unreadCount}</span>
                )}
            </div>

            {loading && notifications.length === 0 && (
                <div className="loading">Загрузка...</div>
            )}

            {notifications.length === 0 && !loading ? (
                <p className="empty-state">Уведомлений нет</p>
            ) : (
                <div className="notifications-list">
                    {notifications.map((notification) => (
                        <div 
                            key={notification.id} 
                            className={`notification-item ${!notification.is_read ? 'unread' : ''}`}
                            onClick={() => !notification.is_read && handleMarkAsRead(notification.id)}
                        >
                            <div className="notification-icon">
                                {getNotificationIcon(notification.notification_type)}
                            </div>
                            <div className="notification-content">
                                <div className="notification-title">
                                    {notification.title}
                                </div>
                                {notification.message && (
                                    <div className="notification-message">
                                        {notification.message}
                                    </div>
                                )}
                                <div className="notification-time">
                                    {formatDate(notification.created_at)}
                                </div>
                            </div>
                            {!notification.is_read && (
                                <div className="read-indicator">●</div>
                            )}
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
};

export default Notifications;
