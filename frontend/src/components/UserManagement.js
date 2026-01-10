import React, { useState, useEffect } from 'react';
import api from '../utils/api';
import { useAuth } from '../contexts/AuthContext';

const UserManagement = () => {
    const [users, setUsers] = useState([]);
    const [loading, setLoading] = useState(false);
    const [savingDepartments, setSavingDepartments] = useState({});
    const [departmentChanges, setDepartmentChanges] = useState({});
    const [deletingUsers, setDeletingUsers] = useState({});
    const [resettingPasswords, setResettingPasswords] = useState({});
    const [passwordRequests, setPasswordRequests] = useState([]);
    const [processingRequests, setProcessingRequests] = useState({});
    const { user: currentUser } = useAuth();

    useEffect(() => {
        fetchUsers();
        if (currentUser?.role === 'admin') fetchPasswordRequests();
    }, []);

    // Run when currentUser becomes available
    React.useEffect(() => {
        if (currentUser?.role === 'admin') fetchPasswordRequests();
    }, [currentUser]);

    const fetchPasswordRequests = async () => {
        try {
            const resp = await api.get('/api/password-reset-requests');
            setPasswordRequests(resp.data.requests || []);
        } catch (err) {
            console.error('Error fetching password requests', err);
        }
    };

    const processPasswordRequest = async (requestId) => {
        if (!window.confirm('Обработать заявку и сгенерировать временный пароль?')) return;
        try {
            setProcessingRequests(prev => ({...prev, [requestId]: true}));
            const resp = await api.post(`/api/password-reset-requests/${requestId}/process`);
            const temp = resp.data.temp_password;
            alert(`Временный пароль для пользователя: ${temp}\nПередайте его пользователю безопасным каналом.`);
            // refresh lists
            fetchPasswordRequests();
            fetchUsers();
        } catch (err) {
            console.error('Error processing request', err);
            alert('Ошибка обработки заявки: ' + (err.response?.data?.error || err.message));
        } finally {
            setProcessingRequests(prev => ({...prev, [requestId]: false}));
        }
    };

    const fetchUsers = async () => {
        try {
            setLoading(true);
            const response = await api.get('/api/users');
            setUsers(response.data);
            setDepartmentChanges({});
        } catch (error) {
            console.error('Error fetching users:', error);
            alert('Error fetching users: ' + (error.response?.data?.error || error.message));
        } finally {
            setLoading(false);
        }
    };

    const updateUserRole = async (userId, newRole) => {
        try {
            setLoading(true);
            await api.put(`/api/users/${userId}/role`, { 
                role: newRole 
            });
            
            setUsers(prevUsers => 
                prevUsers.map(u => 
                    u.id === userId ? { ...u, role: newRole } : u
                )
            );
            
        } catch (error) {
            console.error('Error updating user role:', error);
            alert('Error updating user role: ' + error.response?.data?.error);
        } finally {
            setLoading(false);
        }
    };

    const resetUserPassword = async (userId, username, isADUser) => {
        if (isADUser) {
            alert('Нельзя сбросить пароль пользователю из Active Directory');
            return;
        }

        if (!window.confirm(`Вы уверены, что хотите сбросить пароль пользователю "${username}"?\n\nПользователю будет предложено установить новый пароль при следующем входе.`)) {
            return;
        }

        try {
            setResettingPasswords(prev => ({ ...prev, [userId]: true }));
            
            const response = await api.post(`/api/users/${userId}/reset-password`);
            
            alert(response.data.message || 'Пароль успешно отмечен к сбросу');
            
        } catch (error) {
            console.error('Error resetting password:', error);
            const errorMessage = error.response?.data?.error || 'Unknown error';
            alert('Ошибка сброса пароля: ' + errorMessage);
        } finally {
            setResettingPasswords(prev => ({ ...prev, [userId]: false }));
        }
    };

    const handleDepartmentChange = (userId, newDepartment) => {
        setDepartmentChanges(prev => ({
            ...prev,
            [userId]: newDepartment
        }));
    };

    const saveDepartment = async (userId) => {
        const newDepartment = departmentChanges[userId];
        
        if (!newDepartment || newDepartment.trim() === '') {
            alert('Отдел не может быть пустым');
            return;
        }

        try {
            setSavingDepartments(prev => ({ ...prev, [userId]: true }));
            
            await api.put(`/api/users/${userId}/department`, { 
                department: newDepartment.trim()
            });
            
            setUsers(prevUsers => 
                prevUsers.map(u => 
                    u.id === userId ? { ...u, department: newDepartment.trim() } : u
                )
            );
            
            setDepartmentChanges(prev => {
                const newChanges = { ...prev };
                delete newChanges[userId];
                return newChanges;
            });
            
        } catch (error) {
            console.error('Ошибка в обновлении отдела:', error);
            alert('Ошибка в обновлении отдела: ' + (error.response?.data?.error || 'Unknown error'));
        } finally {
            setSavingDepartments(prev => ({ ...prev, [userId]: false }));
        }
    };

    const cancelDepartmentChange = (userId) => {
        setDepartmentChanges(prev => {
            const newChanges = { ...prev };
            delete newChanges[userId];
            return newChanges;
        });
    };

    const deleteUser = async (userId, username) => {
        if (!window.confirm(`Вы уверены, что хотите удалить пользователя "${username}"? Это действие нельзя отменить.`)) {
            return;
        }

        try {
            setDeletingUsers(prev => ({ ...prev, [userId]: true }));
            
            const response = await api.delete(`/api/users/${userId}`);
            
            alert(response.data.message || `Пользователь ${username} успешно удалён`);
            
            // Удаляем пользователя из списка
            setUsers(prevUsers => prevUsers.filter(u => u.id !== userId));
            
        } catch (error) {
            console.error('Ошибка удаления пользователя:', error);
            const errorMessage = error.response?.data?.error || 'Unknown error';
            alert('Ошибка удаления пользователя: ' + errorMessage);
        } finally {
            setDeletingUsers(prev => ({ ...prev, [userId]: false }));
        }
    };

    const getCurrentDepartment = (userItem) => {
        return departmentChanges[userItem.id] !== undefined 
            ? departmentChanges[userItem.id] 
            : userItem.department || '';
    };

    const hasUnsavedChanges = (userItem) => {
        return departmentChanges[userItem.id] !== undefined;
    };

    const getRoleBadgeClass = (role) => {
        switch (role) {
            case 'admin': return 'role-admin';
            case 'manager': return 'role-manager';
            case 'user': return 'role-user';
            default: return '';
        }
    };

    // Нельзя удалить самого себя или стартового администратора (ID = 1)
    const canDeleteUser = (userItem) => {
        return userItem.id !== currentUser.id && userItem.id !== 1;
    };

    return (
        <div>
            <h2>Настройка пользователей {loading && '(Загрузка...)'}</h2>
            {currentUser?.role === 'admin' && (
                <div style={{marginBottom: '18px', padding: '12px', background: 'var(--card-bg)', border: '1px solid var(--border-color)', borderRadius: '8px'}}>
                    <h3 style={{marginTop:0}}>Заявки на сброс пароля</h3>
                    {passwordRequests.length === 0 ? (
                        <p style={{margin: '8px 0'}}>Нет открытых заявок.</p>
                    ) : (
                        <div style={{display: 'flex', flexDirection: 'column', gap: '8px'}}>
                            {passwordRequests.map(r => (
                                <div key={r.id} style={{display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: '12px'}}>
                                    <div style={{flex:1}}>
                                        <strong>{r.username || 'Неизвестный пользователь'}</strong>
                                        <div style={{color: 'var(--text-secondary)'}}>{r.message}</div>
                                    </div>
                                    <div>
                                        <button className="btn btn-primary" disabled={processingRequests[r.id]} onClick={() => processPasswordRequest(r.id)}>{processingRequests[r.id] ? 'Обработка...' : 'Обработать'}</button>
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            )}
            <div className="users-table">
                <table>
                    <thead>
                        <tr>
                            <th>ID</th>
                            <th>Username</th>
                            <th>Role</th>
                            <th>Department</th>
                            <th>Actions</th>
                            <th>Created At</th>
                        </tr>
                    </thead>
                    <tbody>
                        {users.map(userItem => (
                            <tr key={userItem.id}>
                                <td>{userItem.id}</td>
                                <td>{userItem.username}</td>
                                <td>
                                    <select 
                                        value={userItem.role} 
                                        onChange={(e) => updateUserRole(userItem.id, e.target.value)}
                                        disabled={loading || userItem.id === 1}
                                    >
                                        <option value="user">Сотрудник</option>
                                        <option value="manager">Руководитель</option>
                                        <option value="admin">Админ</option>
                                    </select>
                                    <span className={`role-badge ${getRoleBadgeClass(userItem.role)}`}>
                                        {userItem.role}
                                    </span>
                                </td>
                                <td>
                                    <input
                                        type="text"
                                        value={getCurrentDepartment(userItem)}
                                        onChange={(e) => handleDepartmentChange(userItem.id, e.target.value)}
                                        placeholder="Настройка отдела"
                                        style={{padding: '5px', marginRight: '10px', width: '150px'}}
                                        disabled={loading}
                                    />
                                </td>
                                <td>
                                    <div style={{ display: 'flex', gap: '5px', flexDirection: 'column' }}>
                                        {hasUnsavedChanges(userItem) && (
                                            <div style={{ display: 'flex', gap: '5px' }}>
                                                <button 
                                                    className="btn-save"
                                                    onClick={() => saveDepartment(userItem.id)}
                                                    disabled={savingDepartments[userItem.id]}
                                                    style={{
                                                        padding: '3px 8px',
                                                        fontSize: '12px',
                                                        backgroundColor: '#28a745',
                                                        color: 'white',
                                                        border: 'none',
                                                        borderRadius: '3px',
                                                        cursor: savingDepartments[userItem.id] ? 'not-allowed' : 'pointer'
                                                    }}
                                                >
                                                    {savingDepartments[userItem.id] ? 'Сохранение...' : 'Save'}
                                                </button>
                                                <button 
                                                    className="btn-cancel"
                                                    onClick={() => cancelDepartmentChange(userItem.id)}
                                                    disabled={savingDepartments[userItem.id]}
                                                    style={{
                                                        padding: '3px 8px',
                                                        fontSize: '12px',
                                                        backgroundColor: '#6c757d',
                                                        color: 'white',
                                                        border: 'none',
                                                        borderRadius: '3px',
                                                        cursor: savingDepartments[userItem.id] ? 'not-allowed' : 'pointer'
                                                    }}
                                                >
                                                    Cancel
                                                </button>
                                            </div>
                                        )}
                                        {!userItem.is_ad_user && userItem.id !== 1 && (
                                            <button 
                                                className="btn-reset-password"
                                                onClick={() => resetUserPassword(userItem.id, userItem.username, userItem.is_ad_user)}
                                                disabled={resettingPasswords[userItem.id]}
                                                style={{
                                                    padding: '3px 8px',
                                                    fontSize: '12px',
                                                    backgroundColor: '#ffc107',
                                                    color: '#000',
                                                    border: 'none',
                                                    borderRadius: '3px',
                                                    cursor: resettingPasswords[userItem.id] ? 'not-allowed' : 'pointer',
                                                    marginTop: '5px'
                                                }}
                                            >
                                                {resettingPasswords[userItem.id] ? 'Сброс...' : 'Reset Пароль'}
                                            </button>
                                        )}
                                        {canDeleteUser(userItem) && (
                                            <button 
                                                className="btn-delete"
                                                onClick={() => deleteUser(userItem.id, userItem.username)}
                                                disabled={deletingUsers[userItem.id]}
                                                style={{
                                                    padding: '3px 8px',
                                                    fontSize: '12px',
                                                    backgroundColor: '#dc3545',
                                                    color: 'white',
                                                    border: 'none',
                                                    borderRadius: '3px',
                                                    cursor: deletingUsers[userItem.id] ? 'not-allowed' : 'pointer',
                                                    marginTop: '5px'
                                                }}
                                            >
                                                {deletingUsers[userItem.id] ? 'Удаление...' : 'Delete'}
                                            </button>
                                        )}
                                        {!canDeleteUser(userItem) && (
                                            <span style={{ fontSize: '12px', color: '#6c757d' }}>
                                                Current user
                                            </span>
                                        )}
                                    </div>
                                </td>
                                <td>{new Date(userItem.created_at).toLocaleDateString()}</td>
                            </tr>
                        ))}
                    </tbody>
                </table>
                {users.length === 0 && !loading && <p>Пользователи не найдены.</p>}
            </div>
        </div>
    );
};

export default UserManagement;