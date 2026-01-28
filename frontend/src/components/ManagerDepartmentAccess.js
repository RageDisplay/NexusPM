import React, { useState, useEffect } from 'react';
import api from '../utils/api';

const ManagerDepartmentAccess = ({ managerId, managerName, onClose }) => {
    const [allDepartments, setAllDepartments] = useState([]);
    const [currentDepartments, setCurrentDepartments] = useState([]);
    const [additionalDepartments, setAdditionalDepartments] = useState([]);
    const [selectedDepartment, setSelectedDepartment] = useState('');
    const [loading, setLoading] = useState(false);
    const [adding, setAdding] = useState(false);
    const [removing, setRemoving] = useState({});
    const [error, setError] = useState('');

    useEffect(() => {
        fetchData();
    }, [managerId]);

    const fetchData = async () => {
        try {
            setLoading(true);
            setError('');

            // Получаем все отделы
            const allDeptsResp = await api.get('/api/departments');
            const allDepts = Array.isArray(allDeptsResp.data) ? allDeptsResp.data : allDeptsResp.data?.departments || [];
            setAllDepartments(allDepts);

            // Получаем текущие доступные отделы менеджера
            const currentResp = await api.get(`/api/users/${managerId}/departments`);
            const currentDepts = Array.isArray(currentResp.data) ? currentResp.data : currentResp.data?.departments || [];
            setCurrentDepartments(currentDepts);

            // Получаем дополнительные отделы менеджера
            const additionalResp = await api.get(`/api/users/${managerId}/additional-departments`);
            const additionalDepts = Array.isArray(additionalResp.data) ? additionalResp.data : additionalResp.data?.departments || [];
            setAdditionalDepartments(additionalDepts);

            // Устанавливаем первый доступный отдел для добавления
            if (allDepts && allDepts.length > 0) {
                const firstUnavailable = allDepts.find(
                    dept => !currentDepts.includes(dept)
                );
                if (firstUnavailable) {
                    setSelectedDepartment(firstUnavailable);
                }
            }
        } catch (err) {
            console.error('Error fetching data:', err);
            setError('Ошибка при загрузке данных: ' + (err.response?.data?.error || err.message));
        } finally {
            setLoading(false);
        }
    };

    const handleAddDepartment = async () => {
        if (!selectedDepartment) {
            setError('Пожалуйста, выберите отдел');
            return;
        }

        if (currentDepartments.includes(selectedDepartment)) {
            setError('У менеджера уже есть доступ к этому отделу');
            return;
        }

        try {
            setAdding(true);
            setError('');

            await api.post(`/api/users/${managerId}/departments/add`, {
                department: selectedDepartment
            });

            // Обновляем списки
            await fetchData();
            setSelectedDepartment('');
        } catch (err) {
            console.error('Error adding department:', err);
            setError('Ошибка при добавлении отдела: ' + (err.response?.data?.error || err.message));
        } finally {
            setAdding(false);
        }
    };

    const handleRemoveDepartment = async (department) => {
        if (!window.confirm(`Вы уверены, что хотите удалить доступ к отделу "${department}"?`)) {
            return;
        }

        try {
            setRemoving(prev => ({ ...prev, [department]: true }));
            setError('');

            await api.post(`/api/users/${managerId}/departments/remove`, {
                department
            });

            // Обновляем списки
            await fetchData();
        } catch (err) {
            console.error('Error removing department:', err);
            setError('Ошибка при удалении доступа: ' + (err.response?.data?.error || err.message));
        } finally {
            setRemoving(prev => ({ ...prev, [department]: false }));
        }
    };

    const availableDepartmentsForAdding = allDepartments.filter(
        dept => !currentDepartments.includes(dept)
    );

    return (
        <div style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundColor: 'rgba(0, 0, 0, 0.5)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 1000
        }}>
            <div style={{
                backgroundColor: 'var(--card-bg)',
                borderRadius: '8px',
                padding: '24px',
                maxWidth: '500px',
                width: '90%',
                maxHeight: '90vh',
                overflowY: 'auto',
                boxShadow: '0 4px 6px rgba(0, 0, 0, 0.1)'
            }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
                    <h3 style={{ margin: 0 }}>Управление доступом к отделам</h3>
                    <button
                        onClick={onClose}
                        style={{
                            background: 'none',
                            border: 'none',
                            fontSize: '24px',
                            cursor: 'pointer',
                            color: 'var(--text-secondary)'
                        }}
                    >
                        ×
                    </button>
                </div>

                <p style={{ marginBottom: '16px', color: 'var(--text-secondary)' }}>
                    Менеджер: <strong>{managerName}</strong>
                </p>

                {error && (
                    <div style={{
                        marginBottom: '16px',
                        padding: '12px',
                        backgroundColor: '#fee',
                        border: '1px solid #fcc',
                        borderRadius: '4px',
                        color: '#c00'
                    }}>
                        {error}
                    </div>
                )}

                {loading ? (
                    <p style={{ textAlign: 'center', color: 'var(--text-secondary)' }}>Загрузка...</p>
                ) : (
                    <>
                        {/* Текущие доступные отделы */}
                        <div style={{ marginBottom: '24px' }}>
                            <h4 style={{ marginTop: 0, marginBottom: '12px' }}>Доступные отделы:</h4>
                            {currentDepartments.length === 0 ? (
                                <p style={{ color: 'var(--text-secondary)', margin: 0 }}>Нет доступных отделов</p>
                            ) : (
                                <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                                    {currentDepartments.map((dept) => (
                                        <div
                                            key={dept}
                                            style={{
                                                display: 'flex',
                                                justifyContent: 'space-between',
                                                alignItems: 'center',
                                                padding: '12px',
                                                backgroundColor: 'var(--input-bg)',
                                                borderRadius: '4px',
                                                border: '1px solid var(--border-color)'
                                            }}
                                        >
                                            <span>{dept}</span>
                                            {additionalDepartments.includes(dept) && (
                                                <button
                                                    onClick={() => handleRemoveDepartment(dept)}
                                                    disabled={removing[dept]}
                                                    style={{
                                                        padding: '6px 12px',
                                                        backgroundColor: '#dc3545',
                                                        color: 'white',
                                                        border: 'none',
                                                        borderRadius: '4px',
                                                        cursor: removing[dept] ? 'not-allowed' : 'pointer',
                                                        opacity: removing[dept] ? 0.6 : 1,
                                                        fontSize: '12px'
                                                    }}
                                                >
                                                    {removing[dept] ? 'Удаление...' : 'Удалить'}
                                                </button>
                                            )}
                                        </div>
                                    ))}
                                </div>
                            )}
                        </div>

                        {/* Добавление новых отделов */}
                        <div style={{ marginBottom: '24px' }}>
                            <h4 style={{ marginTop: 0, marginBottom: '12px' }}>Добавить доступ к отделу:</h4>
                            <div style={{ display: 'flex', gap: '8px' }}>
                                <select
                                    value={selectedDepartment}
                                    onChange={(e) => setSelectedDepartment(e.target.value)}
                                    style={{
                                        flex: 1,
                                        padding: '8px 12px',
                                        borderRadius: '4px',
                                        border: '1px solid var(--border-color)',
                                        backgroundColor: 'var(--input-bg)',
                                        color: 'var(--text-primary)',
                                        fontSize: '14px',
                                        colorScheme: 'dark'
                                    }}
                                >
                                    <option value="" style={{ backgroundColor: '#2a2a2a', color: '#fff' }}>-- Выберите отдел --</option>
                                    {availableDepartmentsForAdding.map((dept) => (
                                        <option key={dept} value={dept} style={{ backgroundColor: '#2a2a2a', color: '#fff' }}>
                                            {dept}
                                        </option>
                                    ))}
                                </select>
                                <button
                                    onClick={handleAddDepartment}
                                    disabled={adding || !selectedDepartment}
                                    className="btn btn-primary"
                                    style={{ minWidth: '120px' }}
                                >
                                    {adding ? 'Добавление...' : 'Добавить'}
                                </button>
                            </div>
                        </div>

                        {/* Кнопка закрытия */}
                        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px' }}>
                            <button
                                onClick={onClose}
                                className="btn btn-secondary"
                            >
                                Закрыть
                            </button>
                        </div>
                    </>
                )}
            </div>
        </div>
    );
};

export default ManagerDepartmentAccess;
