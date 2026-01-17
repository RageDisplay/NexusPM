import React, { useState, useEffect } from 'react';
import api from '../utils/api';
import { useAuth } from '../contexts/AuthContext';

const UserProfile = () => {
    const { user, refreshUser } = useAuth();
    const [isEditing, setIsEditing] = useState(false);
    const [formData, setFormData] = useState({
        username: '',
        department: '',
        role: ''
    });
    const [passwordData, setPasswordData] = useState({
        oldPassword: '',
        newPassword: '',
        confirmPassword: ''
    });
    const [showPasswordForm, setShowPasswordForm] = useState(false);
    const [loading, setLoading] = useState(false);
    const [message, setMessage] = useState('');

    useEffect(() => {
        if (user) {
            setFormData({
                username: user.username || '',
                department: user.department || '',
                role: user.role || ''
            });
        }
    }, [user]);

    const handleInputChange = (e) => {
        const { name, value } = e.target;
        setFormData(prev => ({
            ...prev,
            [name]: value
        }));
    };

    const handlePasswordChange = (e) => {
        const { name, value } = e.target;
        setPasswordData(prev => ({
            ...prev,
            [name]: value
        }));
    };

    const handleUpdateDepartment = async () => {
        if (!formData.department.trim()) {
            setMessage('Отдел не может быть пустым');
            return;
        }

        try {
            setLoading(true);
            await api.put('/api/auth/department', {
                department: formData.department.trim()
            });
            setMessage('Отдел успешно обновлён');
            await refreshUser();
            setTimeout(() => setMessage(''), 3000);
        } catch (error) {
            setMessage('Ошибка обновления отдела: ' + (error.response?.data?.error || error.message));
        } finally {
            setLoading(false);
        }
    };

    const handleChangePassword = async () => {
        if (!passwordData.oldPassword || !passwordData.newPassword || !passwordData.confirmPassword) {
            setMessage('Все поля пароля должны быть заполнены');
            return;
        }

        if (passwordData.newPassword !== passwordData.confirmPassword) {
            setMessage('Новые пароли не совпадают');
            return;
        }

        if (passwordData.newPassword.length < 6) {
            setMessage('Пароль должен содержать минимум 6 символов');
            return;
        }

        try {
            setLoading(true);
            await api.post('/api/auth/change-password', {
                current_password: passwordData.oldPassword,
                new_password: passwordData.newPassword
            });
            setMessage('Пароль успешно изменён');
            setShowPasswordForm(false);
            setPasswordData({
                oldPassword: '',
                newPassword: '',
                confirmPassword: ''
            });
            setTimeout(() => setMessage(''), 3000);
        } catch (error) {
            setMessage('Ошибка изменения пароля: ' + (error.response?.data?.error || error.message));
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="user-profile">
            <h2>Мой профиль</h2>
            
            {message && (
                <div className={`message ${message.includes('Ошибка') ? 'error' : 'success'}`}>
                    {message}
                </div>
            )}

            <div className="profile-section">
                <div className="profile-info">
                    <div className="profile-field">
                        <label>Имя пользователя:</label>
                        <input 
                            type="text" 
                            value={formData.username} 
                            disabled
                            className="disabled-field"
                        />
                    </div>

                    <div className="profile-field">
                        <label>Роль:</label>
                        <input 
                            type="text" 
                            value={formData.role} 
                            disabled
                            className="disabled-field"
                        />
                    </div>

                    <div className="profile-field">
                        <label>Отдел:</label>
                        <input 
                            type="text" 
                            name="department"
                            value={formData.department}
                            onChange={handleInputChange}
                            disabled={!isEditing}
                            className={isEditing ? '' : 'disabled-field'}
                        />
                    </div>
                </div>

                <div className="profile-actions">
                    {isEditing ? (
                        <>
                            <button 
                                className="btn btn-primary"
                                onClick={handleUpdateDepartment}
                                disabled={loading}
                            >
                                {loading ? 'Сохранение...' : 'Сохранить'}
                            </button>
                            <button 
                                className="btn btn-secondary"
                                onClick={() => setIsEditing(false)}
                            >
                                Отмена
                            </button>
                        </>
                    ) : (
                        <button 
                            className="btn btn-primary"
                            onClick={() => setIsEditing(true)}
                        >
                            Редактировать профиль
                        </button>
                    )}
                </div>
            </div>

            <div className="password-section">
                {user?.is_ad_user ? (
                    <div className="ad-user-message">
                        <p>Вы авторизованы через Active Directory. Для смены пароля обратитесь к администратору системы.</p>
                    </div>
                ) : !showPasswordForm ? (
                    <button 
                        className="btn btn-secondary"
                        onClick={() => setShowPasswordForm(true)}
                    >
                        Изменить пароль
                    </button>
                ) : (
                    <div className="password-form">
                        <h3>Изменение пароля</h3>
                        <div className="form-group">
                            <label>Текущий пароль:</label>
                            <input 
                                type="password" 
                                name="oldPassword"
                                value={passwordData.oldPassword}
                                onChange={handlePasswordChange}
                                placeholder="Введите текущий пароль"
                            />
                        </div>

                        <div className="form-group">
                            <label>Новый пароль:</label>
                            <input 
                                type="password" 
                                name="newPassword"
                                value={passwordData.newPassword}
                                onChange={handlePasswordChange}
                                placeholder="Введите новый пароль"
                            />
                        </div>

                        <div className="form-group">
                            <label>Подтверждение пароля:</label>
                            <input 
                                type="password" 
                                name="confirmPassword"
                                value={passwordData.confirmPassword}
                                onChange={handlePasswordChange}
                                placeholder="Подтвердите новый пароль"
                            />
                        </div>

                        <div className="form-actions">
                            <button 
                                className="btn btn-primary"
                                onClick={handleChangePassword}
                                disabled={loading}
                            >
                                {loading ? 'Изменение...' : 'Изменить пароль'}
                            </button>
                            <button 
                                className="btn btn-secondary"
                                onClick={() => {
                                    setShowPasswordForm(false);
                                    setPasswordData({
                                        oldPassword: '',
                                        newPassword: '',
                                        confirmPassword: ''
                                    });
                                }}
                            >
                                Отмена
                            </button>
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
};

export default UserProfile;
