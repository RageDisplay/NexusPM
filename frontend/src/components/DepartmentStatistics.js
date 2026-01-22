import React, { useState, useEffect } from 'react';
import api from '../utils/api';
import { useAuth } from '../contexts/AuthContext';

const DepartmentStatistics = () => {
    const { user } = useAuth();
    const [statistics, setStatistics] = useState([]);
    const [loading, setLoading] = useState(false);
    const [clearing, setClearing] = useState(false);
    const [message, setMessage] = useState('');
    const [startDate, setStartDate] = useState('');
    const [endDate, setEndDate] = useState('');

    useEffect(() => {
        if (user?.role === 'manager' || user?.role === 'admin') {
            // Устанавливаем начало недели (понедельник) и конец (воскресенье)
            const today = new Date();
            const dayOfWeek = today.getDay();
            const diff = today.getDate() - dayOfWeek + (dayOfWeek === 0 ? -6 : 1);
            
            const monday = new Date(today.setDate(diff));
            const sunday = new Date(monday);
            sunday.setDate(sunday.getDate() + 6);
            
            const formatDate = (date) => date.toISOString().split('T')[0];
            setStartDate(formatDate(monday));
            setEndDate(formatDate(sunday));
        }
    }, [user]);

    useEffect(() => {
        if (startDate && endDate) {
            fetchStatistics();
        }
    }, [startDate, endDate]);

    const fetchStatistics = async () => {
        try {
            setLoading(true);
            const response = await api.get('/api/department-statistics', {
                params: {
                    start_date: startDate,
                    end_date: endDate
                }
            });
            setStatistics(response.data || []);
            setMessage('');
        } catch (error) {
            console.error('Error fetching statistics:', error);
            setMessage('Ошибка при загрузке статистики');
        } finally {
            setLoading(false);
        }
    };

    const handleClearTasks = async () => {
        const confirmed = window.confirm(
            'Вы уверены? Это удалит все задачи сотрудников вашего отдела.\nЭтого не вернуть!'
        );

        if (!confirmed) {
            return;
        }

        try {
            setClearing(true);
            const response = await api.delete('/api/department-tasks/clear');
            setMessage(`Успешно удалено задач: ${response.data.deleted}`);
            
            // Обновляем статистику после удаления
            setTimeout(() => {
                fetchStatistics();
            }, 500);
        } catch (error) {
            console.error('Error clearing tasks:', error);
            setMessage('Ошибка при удалении задач: ' + (error.response?.data?.error || error.message));
        } finally {
            setClearing(false);
        }
    };

    const getStatusBadge = (hasTasksYet) => {
        if (hasTasksYet) {
            return <span className="status-badge status-has-tasks">✓ Загружены</span>;
        } else {
            return <span className="status-badge status-no-tasks">✗ Не загружены</span>;
        }
    };

    const getRoleColor = (role) => {
        switch (role) {
            case 'admin':
                return '#d32f2f';
            case 'manager':
                return '#f57c00';
            case 'user':
                return '#1976d2';
            default:
                return '#757575';
        }
    };

    const getRoleLabel = (role) => {
        switch (role) {
            case 'admin':
                return 'Админ';
            case 'manager':
                return 'Менеджер';
            case 'user':
                return 'Сотрудник';
            default:
                return role;
        }
    };

    if (user?.role !== 'manager' && user?.role !== 'admin') {
        return null;
    }

    return (
        <div className="department-statistics">
            <div className="statistics-header">
                <h2>Статистика отдела: {user?.department}</h2>
                
                <div className="date-range-selector">
                    <div className="date-input-group">
                        <label htmlFor="start-date">С:</label>
                        <input
                            id="start-date"
                            type="date"
                            value={startDate}
                            onChange={(e) => setStartDate(e.target.value)}
                        />
                    </div>
                    <div className="date-input-group">
                        <label htmlFor="end-date">По:</label>
                        <input
                            id="end-date"
                            type="date"
                            value={endDate}
                            onChange={(e) => setEndDate(e.target.value)}
                        />
                    </div>
                </div>
                
                <div className="header-buttons">
                    <button 
                        onClick={fetchStatistics} 
                        disabled={loading}
                        className="btn btn-refresh"
                    >
                        {loading ? 'Загрузка...' : '🔄 Обновить'}
                    </button>
                    <button 
                        onClick={handleClearTasks}
                        disabled={clearing}
                        className="btn btn-danger"
                    >
                        {clearing ? 'Удаление...' : '🗑️ Очистить задачи'}
                    </button>
                </div>
            </div>

            {message && (
                <div className={`message ${message.includes('Ошибка') ? 'error' : 'success'}`}>
                    {message}
                </div>
            )}

            {loading ? (
                <div className="loading">Загрузка статистики...</div>
            ) : statistics && statistics.length > 0 ? (
                <div className="statistics-grid">
                    {statistics.map((emp) => (
                        <div key={emp.id} className="stat-card">
                            <div className="card-header">
                                <div className="employee-info">
                                    <h3>{emp.username}</h3>
                                    <span 
                                        className="role-badge"
                                        style={{ backgroundColor: getRoleColor(emp.role) }}
                                    >
                                        {getRoleLabel(emp.role)}
                                    </span>
                                </div>
                            </div>

                            <div className="card-body">
                                <div className="stat-row">
                                    <span className="stat-label">Статус отчётности:</span>
                                    <span className="stat-value">
                                        {getStatusBadge(emp.has_tasks)}
                                    </span>
                                </div>

                                <div className="stat-row">
                                    <span className="stat-label">Нагрузка на месяц:</span>
                                    <span className="stat-value load-value">
                                        {emp.total_load_per_month}%
                                    </span>
                                </div>

                                <div className="stat-row">
                                    <span className="stat-label">Часов за неделю:</span>
                                    <span className="stat-value hours-value">
                                        {emp.total_hours_per_week.toFixed(1)} ч.
                                    </span>
                                </div>

                                <div className="stat-row progress-row">
                                    <span className="stat-label">Загруженность:</span>
                                    <div className="progress-bar">
                                        <div 
                                            className="progress-fill"
                                            style={{ 
                                                width: `${Math.min(emp.total_load_per_month, 100)}%`,
                                                backgroundColor: emp.total_load_per_month > 100 ? '#d32f2f' : 
                                                               emp.total_load_per_month > 80 ? '#f57c00' :
                                                               '#4caf50'
                                            }}
                                        >
                                            {emp.total_load_per_month}%
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            ) : (
                <div className="no-data">Нет данных о сотрудниках</div>
            )}
        </div>
    );
};

export default DepartmentStatistics;
