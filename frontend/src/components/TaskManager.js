import React, { useState, useEffect } from 'react';
import api from '../utils/api';
import { useAuth } from '../contexts/AuthContext';

const TaskManager = () => {
    const [tasks, setTasks] = useState([]);
    const [projects, setProjects] = useState([]);
    const [departmentUsers, setDepartmentUsers] = useState([]);
    const [newTask, setNewTask] = useState({
        title: '',
        description: '',
        progress: 0,
        hours_per_week: 0,
        load_per_month: 0,
        project_id: null,
        user_id: 0,
        weekly_info: '',
        planning: '',
        help_needed: ''
    });
    const [loading, setLoading] = useState(false);
    const [editingTask, setEditingTask] = useState(null);
    const { user } = useAuth();

    useEffect(() => {
        fetchProjects();
        fetchTasks();
        if (user?.role === 'manager') {
            fetchDepartmentUsers();
        }
    }, []);

    const fetchProjects = async () => {
        try {
            const response = await api.get('/api/projects/user');
            setProjects(response.data || []);
        } catch (error) {
            console.error('Error fetching projects:', error);
        }
    };

    const fetchDepartmentUsers = async () => {
        try {
            const response = await api.get('/api/users');
            if (response.data) {
                const filtered = response.data.filter(u => u.role === 'user' && u.department === user?.department);
                setDepartmentUsers(filtered);
            }
        } catch (error) {
            console.error('Error fetching department users:', error);
        }
    };

    const fetchTasks = async () => {
        try {
            setLoading(true);
            const response = await api.get('/api/tasks');
            setTasks(response.data);
        } catch (error) {
            console.error('Error fetching tasks:', error);
            alert('Error fetching tasks: ' + (error.response?.data?.error || error.message));
        } finally {
            setLoading(false);
        }
    };

    const handleProjectSelect = (e) => {
        const projectId = parseInt(e.target.value);
        const selectedProject = projects.find(p => p.id === projectId);
        
        setNewTask({
            ...newTask,
            project_id: projectId,
            title: selectedProject?.name || '',
            description: selectedProject?.description || ''
        });
    };

    const createTask = async (e) => {
        e.preventDefault();
        try {
            setLoading(true);
            const response = await api.post('/api/tasks', newTask);
            
            setTasks(prevTasks => [response.data, ...prevTasks]);
            
            await fetchTasks();

            setNewTask({
                title: '',
                description: '',
                progress: 0,
                hours_per_week: 0,
                load_per_month: 0,
                project_id: null,
                user_id: 0,
                weekly_info: '',
                planning: '',
                help_needed: ''
            });
            
        } catch (error) {
            console.error('Error creating task:', error);
            alert('Error creating task: ' + (error.response?.data?.error || error.message));
        } finally {
            setLoading(false);
        }
    };

    const updateProgress = async (taskId, progress) => {
        try {
            const task = tasks.find(t => t.id === taskId);
            await api.put(`/api/tasks/${taskId}`, {
                ...task,
                progress
            });
            
            setTasks(prevTasks => 
                prevTasks.map(t => 
                    t.id === taskId ? { ...t, progress } : t
                )
            );
            
        } catch (error) {
            console.error('Ошибка в изменении шкалы прогресса:', error);
            alert('Ошибка в изменении шкалы прогресса: ' + (error.response?.data?.error || error.message));
        }
    };

    /*const updateHoursPerWeek = async (taskId, hoursPerWeek) => {
        try {
            const task = tasks.find(t => t.id === taskId);
            await api.put(`/api/tasks/${taskId}`, {
                ...task,
                hours_per_week: parseFloat(hoursPerWeek) || 0
            });
            
            setTasks(prevTasks => 
                prevTasks.map(t => 
                    t.id === taskId ? { ...t, hours_per_week: parseFloat(hoursPerWeek) || 0 } : t
                )
            );
            
        } catch (error) {
            console.error('Error updating hours per week:', error);
            alert('Error updating hours per week: ' + (error.response?.data?.error || error.message));
        }
    };

    const updateLoadPerMonth = async (taskId, loadPerMonth) => {
        try {
            const task = tasks.find(t => t.id === taskId);
            await api.put(`/api/tasks/${taskId}`, {
                ...task,
                load_per_month: parseInt(loadPerMonth) || 0
            });
            
            setTasks(prevTasks => 
                prevTasks.map(t => 
                    t.id === taskId ? { ...t, load_per_month: parseInt(loadPerMonth) || 0 } : t
                )
            );
            
        } catch (error) {
            console.error('Error updating load per month:', error);
            alert('Error updating load per month: ' + (error.response?.data?.error || error.message));
        }
    };*/

    const startEditing = (task) => {
        setEditingTask({ ...task });
    };

    const cancelEditing = () => {
        setEditingTask(null);
    };

    const saveTask = async (taskId) => {
        if (!editingTask) return;

        try {
            await api.put(`/api/tasks/${taskId}`, editingTask);
            
            setTasks(prevTasks => 
                prevTasks.map(t => 
                    t.id === taskId ? { ...t, ...editingTask } : t
                )
            );
            
            setEditingTask(null);
        } catch (error) {
            console.error('Ошибка в обновлении задачи:', error);
            alert('Ошибка в обновлении задачи: ' + (error.response?.data?.error || error.message));
        }
    };

    const handleEditChange = (field, value) => {
        setEditingTask(prev => ({
            ...prev,
            [field]: value
        }));
    };

    const deleteTask = async (taskId) => {
        if (!window.confirm('Вы уверены в том, что хотите удалить задачу ?')) {
            return;
        }

        try {
            await api.delete(`/api/tasks/${taskId}`);
            setTasks(prevTasks => prevTasks.filter(t => t.id !== taskId));
        } catch (error) {
            console.error('Ошибка в удалении задачи:', error);
            alert('Ошибка в удалении задачи: ' + (error.response?.data?.error || error.message));
        }
    };

    const canDeleteTask = (task) => {
        if (user.role === 'admin') return true;
        if (user.role === 'manager' && task.department === user.department) return true;
        return task.user_id === user.id;
    };

    const canEditTask = (task) => {
        if (user.role === 'admin') return true;
        if (user.role === 'manager' && task.department === user.department) return true;
        return task.user_id === user.id;
    };

    return (
        <div className="task-manager">
            <h2>Task Management</h2>
            
            <form onSubmit={createTask} className="task-form">
                <h3>Заполнить отчёт</h3>
                
                <div className="form-group">
                    <label>Проект (Название)</label>
                    <select
                        value={newTask.project_id || ''}
                        onChange={handleProjectSelect}
                        required
                        disabled={loading}
                    >
                        <option value="">-- Выберите проект --</option>
                        {projects.map(project => (
                            <option key={project.id} value={project.id}>
                                {project.name}
                            </option>
                        ))}
                    </select>
                </div>

                {user?.role === 'manager' && (
                    <div className="form-group">
                        <label>Назначить сотруднику</label>
                        <select
                            value={newTask.user_id || ''}
                            onChange={(e) => setNewTask({...newTask, user_id: parseInt(e.target.value) || 0})}
                            disabled={loading}
                        >
                            <option value="">-- Выберите сотрудника или оставьте пусто для себя --</option>
                            {departmentUsers.map(deptUser => (
                                <option key={deptUser.id} value={deptUser.id}>
                                    {deptUser.username}
                                </option>
                            ))}
                        </select>
                    </div>
                )}

                {newTask.description && (
                    <div className="project-description-box">
                        <strong>Описание задачи:</strong>
                        <p>{newTask.description}</p>
                    </div>
                )}
                <div className="form-group">
                    <label>Прогресс: {newTask.progress}%</label>
                    <input
                        type="range"
                        min="0"
                        max="100"
                        value={newTask.progress}
                        onChange={(e) => setNewTask({...newTask, progress: parseInt(e.target.value)})}
                        disabled={loading}
                    />
                </div>
                <div className="form-row">
                    <div className="form-group">
                        <label>Часов потрачено</label>
                        <input
                            type="number"
                            step="0.5"
                            min="0"
                            value={newTask.hours_per_week}
                            onChange={(e) => setNewTask({...newTask, hours_per_week: parseFloat(e.target.value) || 0})}
                            disabled={loading}
                        />
                    </div>
                    <div className="form-group">
                        <label>Загрузка с задачи на месяц (%)</label>
                        <input
                            type="number"
                            min="0"
                            max="100"
                            value={newTask.load_per_month}
                            onChange={(e) => setNewTask({...newTask, load_per_month: parseInt(e.target.value) || 0})}
                            disabled={loading}
                        />
                    </div>
                </div>
                <div className="form-row">
                    <div className="form-group">
                        <label>Информация за неделю</label>
                        <textarea
                            placeholder="Что было сделано за неделю"
                            value={newTask.weekly_info}
                            onChange={(e) => setNewTask({...newTask, weekly_info: e.target.value})}
                            disabled={loading}
                        />
                    </div>
                    <div className="form-group">
                        <label>Планирование</label>
                        <textarea
                            placeholder="Что планируется на следующую неделю"
                            value={newTask.planning}
                            onChange={(e) => setNewTask({...newTask, planning: e.target.value})}
                            disabled={loading}
                        />
                    </div>
                </div>
                <div className="form-group">
                    <label>Требуется помощь</label>
                    <input
                        type="text"
                        placeholder="Укажите помощь, если требуется (оставьте пустым, если нет)"
                        value={newTask.help_needed}
                        onChange={(e) => setNewTask({...newTask, help_needed: e.target.value})}
                        disabled={loading}
                    />
                </div>
                <button type="submit" disabled={loading}>
                    {loading ? 'Создание...' : 'Создать'}
                </button>
            </form>

            <div className="tasks-list">
                <h3>Задачи {loading && '(Загрузка...)'}</h3>
                
                {tasks.length === 0 && !loading ? (
                    <p>Задачи не найдены, создайте первую !</p>
                ) : (
                    tasks.map(task => (
                        <div key={task.id} className="task-card">
                            <div className="task-header">
                                {editingTask && editingTask.id === task.id ? (
                                    user.role === 'user' ? (
                                        <h4>{task.title}</h4>
                                    ) : (
                                        <input
                                            type="text"
                                            value={editingTask.title}
                                            onChange={(e) => handleEditChange('title', e.target.value)}
                                            className="edit-input"
                                        />
                                    )
                                ) : (
                                    <h4>{task.title}</h4>
                                )}
                                <div className="task-actions">
                                    {(user.role === 'admin' || user.role === 'manager') && (
                                        <span className="task-meta">by {task.username} ({task.department})</span>
                                    )}
                                    {canEditTask(task) && (
                                        <>
                                            {editingTask && editingTask.id === task.id ? (
                                                <>
                                                    <button 
                                                        className="btn-save"
                                                        onClick={() => saveTask(task.id)}
                                                    >
                                                        Сохранить
                                                    </button>
                                                    <button 
                                                        className="btn-cancel"
                                                        onClick={cancelEditing}
                                                    >
                                                        Отмена
                                                    </button>
                                                </>
                                            ) : (
                                                <button 
                                                    className="btn-edit"
                                                    onClick={() => startEditing(task)}
                                                >
                                                    Изменить
                                                </button>
                                            )}
                                        </>
                                    )}
                                    {canDeleteTask(task) && (
                                        <button 
                                            className="delete-btn"
                                            onClick={() => deleteTask(task.id)}
                                            disabled={loading}
                                        >
                                            Удалить
                                        </button>
                                    )}
                                </div>
                            </div>
                            
                            {editingTask && editingTask.id === task.id ? (
                                user.role === 'user' ? (
                                    <p>{task.description}</p>
                                ) : (
                                    <textarea
                                        value={editingTask.description}
                                        onChange={(e) => handleEditChange('description', e.target.value)}
                                        className="edit-textarea"
                                    />
                                )
                            ) : (
                                <p>{task.description}</p>
                            )}
                            
                            <div className="task-progress">
                                <label>Прогресс: {task.progress}%</label>
                                <input
                                    type="range"
                                    min="0"
                                    max="100"
                                    value={task.progress}
                                    onChange={(e) => updateProgress(task.id, parseInt(e.target.value))}
                                    disabled={loading}
                                />
                            </div>
                            
                            <div className="task-stats">
                                <div className="stat-item">
                                    <label>Часов потрачено: </label>
                                    {editingTask && editingTask.id === task.id ? (
                                        <input
                                            type="number"
                                            step="0.5"
                                            min="0"
                                            value={editingTask.hours_per_week}
                                            onChange={(e) => handleEditChange('hours_per_week', parseFloat(e.target.value) || 0)}
                                            className="edit-number"
                                        />
                                    ) : (
                                        <span 
                                            className="editable-field"
                                            onClick={() => canEditTask(task) && startEditing(task)}
                                            title={canEditTask(task) ? "Нажмите для редактирования" : ""}
                                        >
                                            {task.hours_per_week}
                                        </span>
                                    )}
                                </div>
                                
                                <div className="stat-item">
                                    <label>Нагрузка на месяц: </label>
                                    {editingTask && editingTask.id === task.id ? (
                                        <input
                                            type="number"
                                            min="0"
                                            max="100"
                                            value={editingTask.load_per_month}
                                            onChange={(e) => handleEditChange('load_per_month', parseInt(e.target.value) || 0)}
                                            className="edit-number"
                                        />
                                    ) : (
                                        <span 
                                            className="editable-field"
                                            onClick={() => canEditTask(task) && startEditing(task)}
                                            title={canEditTask(task) ? "Нажмите для редактирования" : ""}
                                        >
                                            {task.load_per_month}%
                                        </span>
                                    )}
                                </div>
                                
                                <div className="stat-item">
                                    <span>Создана: {new Date(task.created_at).toLocaleDateString()}</span>
                                </div>
                            </div>

                            <div className="task-report-section">
                                <div className="report-item">
                                    <label>Информация за неделю:</label>
                                    {editingTask && editingTask.id === task.id ? (
                                        <textarea
                                            value={editingTask.weekly_info || ''}
                                            onChange={(e) => handleEditChange('weekly_info', e.target.value)}
                                            className="edit-textarea"
                                        />
                                    ) : (
                                        <p className="report-text">{task.weekly_info || 'Нет информации'}</p>
                                    )}
                                </div>

                                <div className="report-item">
                                    <label>Планирование:</label>
                                    {editingTask && editingTask.id === task.id ? (
                                        <textarea
                                            value={editingTask.planning || ''}
                                            onChange={(e) => handleEditChange('planning', e.target.value)}
                                            className="edit-textarea"
                                        />
                                    ) : (
                                        <p className="report-text">{task.planning || 'Нет информации'}</p>
                                    )}
                                </div>

                                <div className="report-item">
                                    <label>Требуется помощь:</label>
                                    {editingTask && editingTask.id === task.id ? (
                                        <input
                                            type="text"
                                            value={editingTask.help_needed || ''}
                                            onChange={(e) => handleEditChange('help_needed', e.target.value)}
                                            className="edit-input"
                                        />
                                    ) : (
                                        <p className="report-text">{task.help_needed || 'Нет'}</p>
                                    )}
                                </div>
                            </div>
                        </div>
                    ))
                )}
            </div>
        </div>
    );
};

export default TaskManager;