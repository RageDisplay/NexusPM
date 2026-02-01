import React, { useState, useEffect } from 'react';
import api from '../utils/api';
import { useAuth } from '../contexts/AuthContext';

const ProjectManagement = () => {
    const [projects, setProjects] = useState([]);
    const [users, setUsers] = useState([]);
    const [newProject, setNewProject] = useState({
        name: '',
        description: ''
    });
    const [selectedProject, setSelectedProject] = useState(null);
    const [selectedUserId, setSelectedUserId] = useState('');
    const [selectedUserIds, setSelectedUserIds] = useState(new Set());
    const [projectUsers, setProjectUsers] = useState([]);
    const [loading, setLoading] = useState(false);
    const { user } = useAuth();

    useEffect(() => {
        fetchProjects();
        fetchUsers();
    }, [user]);

    const fetchProjects = async () => {
        try {
            setLoading(true);
            // Все пользователи (админ и менеджер) видят все проекты
            const response = await api.get('/api/projects');
            const projectList = response.data || [];
            setProjects(projectList);
        } catch (error) {
            console.error('Error fetching projects:', error);
        } finally {
            setLoading(false);
        }
    };

    const fetchUsers = async () => {
        try {
            const response = await api.get('/api/users');
            console.log('All users from API:', response.data);
            if (response.data) {
                let userList = response.data;
                
                // Для менеджера - пользователи всех доступных ему отделов (первичный + дополнительные)
                if (user?.role === 'manager') {
                    try {
                        const managerDepartmentsResponse = await api.get(`/api/users/${user.id}/departments`);
                        console.log('Manager departments response:', managerDepartmentsResponse.data);
                        const managerDepartments = Array.isArray(managerDepartmentsResponse.data) 
                            ? managerDepartmentsResponse.data 
                            : managerDepartmentsResponse.data?.departments || [];
                        console.log('Parsed manager departments for user', user.id, ':', managerDepartments);
                        userList = userList.filter(u => managerDepartments.includes(u.department));
                        console.log('Filtered users by departments:', userList);
                    } catch (error) {
                        console.warn('Could not fetch manager departments:', error);
                        console.log('Fallback: using user department:', user?.department);
                        userList = userList.filter(u => u.department === user?.department);
                        console.log('Filtered users by single department:', userList);
                    }
                }
                // Для админа - все пользователи
                
                setUsers(userList);
            }
        } catch (error) {
            console.error('Error fetching users:', error);
        }
    };

    const fetchProjectUsers = async (projectId) => {
        try {
            const response = await api.get(`/api/projects/${projectId}/users`);
            setProjectUsers(response.data || []);
        } catch (error) {
            console.error('Error fetching project users:', error);
        }
    };

    const handleCreateProject = async (e) => {
        e.preventDefault();
        if (!newProject.name.trim()) {
            alert('Название проекта не должно быть пустым');
            return;
        }
        try {
            setLoading(true);
            await api.post('/api/projects', newProject);
            setNewProject({ name: '', description: '' });
            await fetchProjects();
            alert('Проект успешно создан!');
        } catch (error) {
            console.error('Error creating project:', error);
            alert('Ошибка при создании проекта: ' + (error.response?.data?.error || error.message));
        } finally {
            setLoading(false);
        }
    };

    const handleSelectProject = async (project) => {
        setSelectedProject(project);
        await fetchProjectUsers(project.id);
    };

    const handleAssignUserToProject = async (userId, projectId) => {
        if (!userId) {
            alert('Выберите сотрудника');
            return;
        }
        try {
            setLoading(true);
            await api.post('/api/projects/assign', {
                user_id: userId,
                project_id: projectId
            });
            await fetchProjectUsers(projectId);
            setSelectedUserId('');
            alert('Проект успешно назначен!');
        } catch (error) {
            console.error('Error assigning project:', error);
            alert('Ошибка: ' + (error.response?.data?.error || error.message));
        } finally {
            setLoading(false);
        }
    };

    const handleAssignMultipleUsers = async (projectId) => {
        if (selectedUserIds.size === 0) {
            alert('Выберите хотя бы одного сотрудника');
            return;
        }

        try {
            setLoading(true);
            const userIdArray = Array.from(selectedUserIds);
            let successCount = 0;
            let errorCount = 0;

            for (const userId of userIdArray) {
                try {
                    await api.post('/api/projects/assign', {
                        user_id: userId,
                        project_id: projectId
                    });
                    successCount++;
                } catch (error) {
                    console.error(`Error assigning user ${userId}:`, error);
                    errorCount++;
                }
            }

            await fetchProjectUsers(projectId);
            setSelectedUserIds(new Set());

            let message = `✓ Добавлено сотрудников: ${successCount}`;
            if (errorCount > 0) {
                message += `\n⚠ Ошибок: ${errorCount}`;
            }
            alert(message);
        } catch (error) {
            console.error('Error in batch assignment:', error);
            alert('Ошибка при массовом добавлении: ' + error.message);
        } finally {
            setLoading(false);
        }
    };

    const toggleUserSelection = (userId) => {
        const newSelected = new Set(selectedUserIds);
        if (newSelected.has(userId)) {
            newSelected.delete(userId);
        } else {
            newSelected.add(userId);
        }
        setSelectedUserIds(newSelected);
    };

    const toggleSelectAll = () => {
        const availableUsers = users.filter(u => !projectUsers.some(pu => pu.id === u.id));
        if (selectedUserIds.size === availableUsers.length) {
            setSelectedUserIds(new Set());
        } else {
            setSelectedUserIds(new Set(availableUsers.map(u => u.id)));
        }
    };

    const handleRemoveUserFromProject = async (userId, projectId) => {
        if (!window.confirm('Удалить проект у этого сотрудника?')) {
            return;
        }
        try {
            setLoading(true);
            await api.post('/api/projects/remove', {
                user_id: userId,
                project_id: projectId
            });
            await fetchProjectUsers(projectId);
            alert('Проект удалён!');
        } catch (error) {
            console.error('Error removing project:', error);
            alert('Ошибка: ' + (error.response?.data?.error || error.message));
        } finally {
            setLoading(false);
        }
    };

    const handleDeleteProject = async (projectId) => {
        if (!window.confirm('Вы действительно хотите удалить этот проект? Это действие необратимо.')) {
            return;
        }
        try {
            setLoading(true);
            await api.delete(`/api/projects/${projectId}`);
            await fetchProjects();
            setSelectedProject(null);
            setSelectedUserId('');
            alert('Проект успешно удалён!');
        } catch (error) {
            console.error('Error deleting project:', error);
            alert('Ошибка при удалении проекта: ' + (error.response?.data?.error || error.message));
        } finally {
            setLoading(false);
        }
    };

    if (user?.role !== 'admin' && user?.role !== 'manager') {
        return (
            <div className="projects-container">
                <h2>Управление проектами</h2>
                <p style={{ color: '#e74c3c', fontSize: '1.1rem' }}>У вас нет прав доступа к этому разделу.</p>
            </div>
        );
    }

    return (
        <div className="projects-container">
            <h2>Проекты</h2>
            
            {/* Create New Project */}
            <div className="projects-create-section">
                <h3>Создать новый проект</h3>
                <form className="projects-create-form" onSubmit={handleCreateProject}>
                    <input
                        type="text"
                        placeholder="Название проекта"
                        value={newProject.name}
                        onChange={(e) => setNewProject({ ...newProject, name: e.target.value })}
                        maxLength="255"
                        disabled={loading}
                    />
                    <textarea
                        placeholder="Описание проекта (детали, требования, ожидаемые результаты)"
                        value={newProject.description}
                        onChange={(e) => setNewProject({ ...newProject, description: e.target.value })}
                        maxLength="2000"
                        disabled={loading}
                    />
                    <button className="btn btn-primary" type="submit" disabled={loading}>
                        {loading ? 'Создание...' : '✓ Создать проект'}
                    </button>
                </form>
            </div>

            {/* Project List and Management */}
            <div className="projects-manage-section">
                {/* Projects List */}
                <div className="projects-list">
                    <h4>Список проектов ({projects.length})</h4>
                    {projects.length === 0 ? (
                        <div className="loading-message">Проектов не найдено</div>
                    ) : (
                        projects.map(project => (
                            <div
                                key={project.id}
                                className={`project-item ${selectedProject?.id === project.id ? 'selected' : ''}`}
                                onClick={() => handleSelectProject(project)}
                            >
                                <div className="project-item-name">{project.name}</div>
                                <div className="project-item-desc">{project.description}</div>
                                <div className="project-item-dept">Отдел: {project.department}</div>
                            </div>
                        ))
                    )}
                </div>

                {/* Project Details and User Assignment */}
                <div className={`project-details ${!selectedProject ? 'empty' : ''}`}>
                    {selectedProject ? (
                        <>
                            <h4>Детали проекта</h4>

                            <div className="project-detail-item">
                                <label>Название:</label>
                                <div className="project-detail-value">{selectedProject.name}</div>
                            </div>

                            <div className="project-detail-item">
                                <label>Описание:</label>
                                <div className="project-detail-value">{selectedProject.description}</div>
                            </div>

                            <div className="project-detail-item">
                                <label>Отдел:</label>
                                <div className="project-detail-value">{selectedProject.department}</div>
                            </div>

                            <div className="project-detail-item">
                                <button 
                                    className="btn btn-danger"
                                    onClick={() => handleDeleteProject(selectedProject.id)}
                                    disabled={loading}
                                >
                                    🗑️ Удалить проект
                                </button>
                            </div>

                            {/* Assigned Users */}
                            <div className="project-users">
                                <h5>Назначенные сотрудники ({projectUsers.length})</h5>
                                {projectUsers.length === 0 ? (
                                    <div className="loading-message">Сотрудники не назначены</div>
                                ) : (
                                    <div className="project-users-list">
                                        {projectUsers.map(projectUser => (
                                            <div key={projectUser.id} className="project-user-item">
                                                <div className="project-user-info">
                                                    <span className="project-user-name">{projectUser.username}</span>
                                                    <span className="project-user-department">({projectUser.department})</span>
                                                </div>
                                                <button
                                                    className="btn-remove-user"
                                                    onClick={() => handleRemoveUserFromProject(projectUser.id, selectedProject.id)}
                                                    disabled={loading}
                                                >
                                                    ✕ Удалить
                                                </button>
                                            </div>
                                        ))}
                                    </div>
                                )}

                                {/* Assign User to Project */}
                                <div className="assign-users-section">
                                    <h5>Добавить сотрудников</h5>

                                    {/* Batch assignment mode */}
                                    <div className="batch-assign-container" style={{ border: '1px solid #ddd', borderRadius: '4px', padding: '15px' }}>
                                        <div className="users-checkbox-list" style={{ maxHeight: '300px', overflowY: 'auto' }}>
                                            {users.filter(u => !projectUsers.some(pu => pu.id === u.id)).length === 0 ? (
                                                <div className="loading-message">Все доступные сотрудники уже добавлены на проект</div>
                                            ) : (
                                                <>
                                                    <div className="checkbox-item" style={{ 
                                                        padding: '10px', 
                                                        backgroundColor: 'rgba(255, 159, 67, 0.15)',
                                                        borderRadius: '4px',
                                                        marginBottom: '10px',
                                                        border: '1px solid rgba(255, 159, 67, 0.3)'
                                                    }}>
                                                        <label style={{ display: 'flex', alignItems: 'center', gap: '10px', cursor: 'pointer' }}>
                                                            <input 
                                                                type="checkbox"
                                                                checked={selectedUserIds.size > 0 && selectedUserIds.size === users.filter(u => !projectUsers.some(pu => pu.id === u.id)).length}
                                                                onChange={toggleSelectAll}
                                                                disabled={loading}
                                                            />
                                                            <strong>Выбрать всех ({users.filter(u => !projectUsers.some(pu => pu.id === u.id)).length})</strong>
                                                        </label>
                                                    </div>

                                                    <div style={{ borderTop: '1px solid #ddd', paddingTop: '10px' }}>
                                                        {users
                                                            .filter(u => !projectUsers.some(pu => pu.id === u.id))
                                                            .map(u => (
                                                                <label 
                                                                    key={u.id}
                                                                    className="checkbox-item" 
                                                                    style={{ 
                                                                        display: 'flex', 
                                                                        alignItems: 'center',
                                                                        gap: '10px',
                                                                        padding: '8px',
                                                                        cursor: 'pointer',
                                                                        borderRadius: '4px',
                                                                        backgroundColor: selectedUserIds.has(u.id) ? '#ff9d0045' : 'transparent',
                                                                        marginBottom: '5px'
                                                                    }}
                                                                >
                                                                    <input 
                                                                        type="checkbox"
                                                                        checked={selectedUserIds.has(u.id)}
                                                                        onChange={() => toggleUserSelection(u.id)}
                                                                        disabled={loading}
                                                                    />
                                                                    <span>{u.username}</span>
                                                                    <span style={{ color: '#999', fontSize: '0.9em' }}>({u.department})</span>
                                                                </label>
                                                            ))}
                                                    </div>
                                                </>
                                            )}
                                        </div>

                                        <div style={{ marginTop: '15px', display: 'flex', gap: '10px', justifyContent: 'center' }}>
                                            <button
                                                className="btn btn-primary"
                                                onClick={() => handleAssignMultipleUsers(selectedProject.id)}
                                                disabled={selectedUserIds.size === 0 || loading}
                                                style={{ flex: 1 }}
                                            >
                                                {loading ? 'Добавление...' : `✓ Добавить выбранных (${selectedUserIds.size})`}
                                            </button>
                                            <button
                                                className="btn btn-secondary"
                                                onClick={() => setSelectedUserIds(new Set())}
                                                disabled={loading}
                                                style={{ flex: 1 }}
                                            >
                                                ✕ Очистить выбор
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </>
                    ) : (
                        'Выберите проект для просмотра деталей'
                    )}
                </div>
            </div>
        </div>
    );
};

export default ProjectManagement;
