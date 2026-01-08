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
            if (response.data) {
                let userList = response.data.filter(u => u.role === 'user');
                
                // Для менеджера - только пользователи его отдела (для назначения)
                if (user?.role === 'manager') {
                    userList = userList.filter(u => u.department === user?.department);
                }
                
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
                        disabled={loading}
                    />
                    <textarea
                        placeholder="Описание проекта (детали, требования, ожидаемые результаты)"
                        value={newProject.description}
                        onChange={(e) => setNewProject({ ...newProject, description: e.target.value })}
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
                                    <h5>Добавить сотрудника</h5>
                                    <form className="assign-users-form" onSubmit={(e) => {
                                        e.preventDefault();
                                        handleAssignUserToProject(selectedUserId, selectedProject.id);
                                    }}>
                                        <select
                                            value={selectedUserId}
                                            onChange={(e) => setSelectedUserId(parseInt(e.target.value) || '')}
                                        >
                                            <option value="">-- Выберите сотрудника --</option>
                                            {/* Для менеджера и админа добавляем их самих в список */}
                                            {(user?.role === 'manager' || user?.role === 'admin') && (
                                                <option value={user.id}>
                                                    {user.username} ({user.department}) - Я
                                                </option>
                                            )}
                                            {users
                                                .filter(u => !projectUsers.some(pu => pu.id === u.id))
                                                .map(u => (
                                                    <option key={u.id} value={u.id}>
                                                        {u.username} ({u.department})
                                                    </option>
                                                ))}
                                        </select>
                                        <button
                                            type="submit"
                                            className="btn-assign"
                                            disabled={!selectedUserId || loading}
                                        >
                                            {loading ? 'Добавление...' : 'Добавить'}
                                        </button>
                                    </form>
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
