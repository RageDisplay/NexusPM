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

    // Фильтрация и поиск
    const [searchQuery, setSearchQuery] = useState('');
    const [filterProject, setFilterProject] = useState('');
    const [filterStatus, setFilterStatus] = useState(''); // 'active', 'completed'
    const [filterUser, setFilterUser] = useState(''); // Фильтр по ФИО сотрудника
    const [sortBy, setSortBy] = useState('date'); // 'date', 'progress', 'hours'

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
            console.log('All users from API:', response.data);
            if (response.data) {
                try {
                    // Получаем доступные отделы менеджера
                    const deptResponse = await api.get(`/api/users/${user.id}/departments`);
                    console.log('Manager departments response:', deptResponse.data);
                    const managerDepartments = Array.isArray(deptResponse.data) 
                        ? deptResponse.data 
                        : deptResponse.data?.departments || [];
                    
                    console.log('Parsed manager departments:', managerDepartments);
                    
                    // Фильтруем пользователей по доступным отделам
                    const filtered = response.data.filter(u => 
                        managerDepartments.includes(u.department)
                    );
                    console.log('Filtered department users:', filtered);
                    setDepartmentUsers(filtered);
                } catch (error) {
                    // Если не удалось получить доступные отделы, используем только основной отдел
                    console.warn('Could not fetch manager departments:', error);
                    console.log('User department:', user?.department);
                    const filtered = response.data.filter(u => u.department === user?.department);
                    console.log('Fallback filtered users:', filtered);
                    setDepartmentUsers(filtered);
                }
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

    // Функция для фильтрации и сортировки задач
    const getFilteredAndSortedTasks = () => {
        let filtered = tasks;

        // Поиск
        if (searchQuery) {
            filtered = filtered.filter(task => 
                task.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
                task.description.toLowerCase().includes(searchQuery.toLowerCase())
            );
        }

        // Фильтр по проекту
        if (filterProject) {
            filtered = filtered.filter(task => task.project_id === parseInt(filterProject));
        }

        // Фильтр по статусу
        if (filterStatus === 'active') {
            filtered = filtered.filter(task => task.progress < 100);
        } else if (filterStatus === 'completed') {
            filtered = filtered.filter(task => task.progress === 100);
        }

        // Фильтр по пользователю/сотруднику
        if (filterUser) {
            filtered = filtered.filter(task => {
                // Ищем пользователя по введённому имени
                const selectedUser = departmentUsers.find(u => {
                    const userFullName = u.last_name && u.first_name 
                        ? `${u.last_name} ${u.first_name}${u.patronymic ? ' ' + u.patronymic : ''}`
                        : u.username;
                    return userFullName === filterUser;
                });
                
                // Если нашли пользователя, проверяем, что задача ему принадлежит
                if (selectedUser) {
                    return task.user_id === selectedUser.id;
                }
                return false;
            });
        }

        // Сортировка
        if (sortBy === 'progress') {
            filtered.sort((a, b) => b.progress - a.progress);
        } else if (sortBy === 'hours') {
            filtered.sort((a, b) => b.hours_per_week - a.hours_per_week);
        } else { // date
            filtered.sort((a, b) => new Date(b.created_at) - new Date(a.created_at));
        }

        return filtered;
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

    const [localProgress, setLocalProgress] = useState({});

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

    const handleProgressChange = (taskId, value) => {
        // Обновляем локальное состояние для визуального обновления
        setLocalProgress(prev => ({
            ...prev,
            [taskId]: parseInt(value)
        }));
    };

    const handleProgressCommit = (taskId, value) => {
        // Отправляем на сервер только при отпускании ползунка
        updateProgress(taskId, parseInt(value));
        // Очищаем локальное состояние
        setLocalProgress(prev => {
            const updated = { ...prev };
            delete updated[taskId];
            return updated;
        });
    };

    const duplicateTask = async (task) => {
        try {
            setLoading(true);
            const response = await api.post(`/api/tasks/${task.id}/duplicate`);
            setTasks(prevTasks => [response.data.task, ...prevTasks]);
            alert('Задача успешно дублирована!');
        } catch (error) {
            console.error('Error duplicating task:', error);
            alert('Ошибка при дублировании задачи: ' + (error.response?.data?.error || error.message));
        } finally {
            setLoading(false);
        }
    };

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

    const canDuplicateTask = (task) => {
        if (user.role === 'admin') return true;
        if (user.role === 'manager') return task.department === user.department;
        return task.user_id === user.id;
    };

    const filteredTasks = getFilteredAndSortedTasks();

    return (
        <div className="task-manager">
            <h2>Управление задачами</h2>
            
            <form onSubmit={createTask} className="task-form">
                <h3>Заполнить отчёт</h3>
                
                <div className="form-group form-col-1">
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
                    <div className="form-group form-col-2">
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

                <div className="project-description-box" style={{opacity: newTask.description ? 1 : 0.3}}>
                    <strong>Описание задачи:</strong>
                    <p>{newTask.description || 'Выберите проект для отображения описания'}</p>
                </div>

                <div className="form-group form-col-1">
                    <label>Информация за неделю</label>
                    <textarea
                        placeholder="Что было сделано за неделю"
                        value={newTask.weekly_info}
                        onChange={(e) => setNewTask({...newTask, weekly_info: e.target.value})}
                        disabled={loading}
                    />
                </div>

                <div className="form-group form-col-1">
                    <textarea
                        placeholder="Что планируется на следующую неделю"
                        value={newTask.planning}
                        onChange={(e) => setNewTask({...newTask, planning: e.target.value})}
                        disabled={loading}
                    />
                </div>

                <div className="form-group progress-group">
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

                <div className="form-group form-col-1">
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

                <div className="form-group form-col-2">
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

                <div className="form-group form-col-3">
                    <label>Требуется помощь</label>
                    <input
                        type="text"
                        placeholder="Укажите помощь, если требуется"
                        value={newTask.help_needed}
                        onChange={(e) => setNewTask({...newTask, help_needed: e.target.value})}
                        disabled={loading}
                    />
                </div>

                <button type="submit" className="btn btn-primary" disabled={loading}>
                    {loading ? 'Создание...' : 'Создать'}
                </button>
            </form>

            <div className="tasks-list">
                <h3>Задачи {loading && '(Загрузка...)'}</h3>
                
                {/* Фильтры и поиск */}
                <div className="tasks-controls">
                    <div className="search-box">
                        <input
                            type="text"
                            placeholder="🔍 Поиск по названию или описанию..."
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            className="search-input"
                        />
                    </div>

                    <div className="filters-row">
                        <div className="filter-group">
                            <label>Проект:</label>
                            <select
                                value={filterProject}
                                onChange={(e) => setFilterProject(e.target.value)}
                                className="filter-select"
                            >
                                <option value="">Все проекты</option>
                                {projects.map(project => (
                                    <option key={project.id} value={project.id}>
                                        {project.name}
                                    </option>
                                ))}
                            </select>
                        </div>

                        <div className="filter-group">
                            <label>Статус:</label>
                            <select
                                value={filterStatus}
                                onChange={(e) => setFilterStatus(e.target.value)}
                                className="filter-select"
                            >
                                <option value="">Все статусы</option>
                                <option value="active">Активные</option>
                                <option value="completed">Завершённые</option>
                            </select>
                        </div>

                        {(user?.role === 'manager' || user?.role === 'admin') && departmentUsers.length > 0 && (
                            <div className="filter-group">
                                <label>Сотрудник:</label>
                                <select
                                    value={filterUser}
                                    onChange={(e) => setFilterUser(e.target.value)}
                                    className="filter-select"
                                >
                                    <option value="">Все сотрудники</option>
                                    {departmentUsers.map(u => {
                                        const fullName = u.last_name && u.first_name 
                                            ? `${u.last_name} ${u.first_name}${u.patronymic ? ' ' + u.patronymic : ''}`
                                            : u.username;
                                        return (
                                            <option key={u.id} value={fullName}>
                                                {fullName}
                                            </option>
                                        );
                                    })}
                                </select>
                            </div>
                        )}

                        <div className="filter-group">
                            <label>Сортировка:</label>
                            <select
                                value={sortBy}
                                onChange={(e) => setSortBy(e.target.value)}
                                className="filter-select"
                            >
                                <option value="date">По дате (новые первыми)</option>
                                <option value="progress">По прогрессу (выше первыми)</option>
                                <option value="hours">По часам (больше первыми)</option>
                            </select>
                        </div>
                    </div>

                    <div className="results-info">
                        Найдено задач: <strong>{filteredTasks.length}</strong>
                    </div>
                </div>
                
                {filteredTasks.length === 0 && !loading ? (
                    <p className="empty-state">Задачи не найдены{searchQuery ? ' по вашему запросу' : ', создайте первую!'}</p>
                ) : (
                    <div className="tasks-grid">
                        {filteredTasks.map(task => (
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
                                        {canDuplicateTask(task) && (
                                            <button 
                                                className="btn-duplicate"
                                                onClick={() => duplicateTask(task)}
                                                disabled={loading}
                                                title="Дублировать задачу на следующую неделю"
                                            >
                                                Дублировать
                                            </button>
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
                                    <label>Прогресс: {localProgress[task.id] !== undefined ? localProgress[task.id] : task.progress}%</label>
                                    <input
                                        type="range"
                                        min="0"
                                        max="100"
                                        value={localProgress[task.id] !== undefined ? localProgress[task.id] : task.progress}
                                        onChange={(e) => handleProgressChange(task.id, e.target.value)}
                                        onMouseUp={(e) => handleProgressCommit(task.id, e.target.value)}
                                        onTouchEnd={(e) => handleProgressCommit(task.id, e.target.value)}
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
                        ))}
                    </div>
                )}
            </div>
        </div>
    );
};

export default TaskManager;