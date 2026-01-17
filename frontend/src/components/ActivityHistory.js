import React, { useState, useEffect } from 'react';
import api from '../utils/api';

const ActivityHistory = () => {
    const [activities, setActivities] = useState([]);
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        fetchActivities();
    }, []);

    const fetchActivities = async () => {
        try {
            setLoading(true);
            const response = await api.get('/api/dashboard/activity');
            setActivities(response.data || []);
        } catch (error) {
            console.error('Error fetching activities:', error);
        } finally {
            setLoading(false);
        }
    };

    const getActivityIcon = (actionType) => {
        switch (actionType) {
            case 'task_created':
                return '✨';
            case 'task_updated':
                return '✏️';
            case 'task_completed':
                return '✅';
            case 'project_assigned':
                return '📋';
            case 'profile_updated':
                return '👤';
            default:
                return '📝';
        }
    };

    const getActivityLabel = (actionType) => {
        const labels = {
            'task_created': 'Задача создана',
            'task_updated': 'Задача обновлена',
            'task_completed': 'Задача завершена',
            'project_assigned': 'Проект назначен',
            'profile_updated': 'Профиль обновлён',
            'login': 'Вход в систему',
        };
        return labels[actionType] || actionType;
    };

    const formatDate = (dateString) => {
        const date = new Date(dateString);
        return date.toLocaleDateString('ru-RU', {
            day: '2-digit',
            month: '2-digit',
            year: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });
    };

    if (loading) {
        return <div className="activity-history loading">Загрузка...</div>;
    }

    return (
        <div className="activity-history">
            <h3>История активностей</h3>
            
            {activities.length === 0 ? (
                <p className="empty-state">Нет активностей</p>
            ) : (
                <div className="activity-list">
                    {activities.map((activity) => (
                        <div key={activity.id} className="activity-item">
                            <div className="activity-icon">
                                {getActivityIcon(activity.action_type)}
                            </div>
                            <div className="activity-content">
                                <div className="activity-action">
                                    {getActivityLabel(activity.action_type)}
                                </div>
                                {activity.description && (
                                    <div className="activity-description">
                                        {activity.description}
                                    </div>
                                )}
                                <div className="activity-time">
                                    {formatDate(activity.created_at)}
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
};

export default ActivityHistory;
