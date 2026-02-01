import React, { useState, useEffect } from 'react';
import api from '../utils/api';
import { useAuth } from '../contexts/AuthContext';

const MyProjects = () => {
    const { user } = useAuth();
    const [myProjects, setMyProjects] = useState([]);
    const [availableProjects, setAvailableProjects] = useState([]);
    const [loading, setLoading] = useState(false);
    const [draggedProject, setDraggedProject] = useState(null);
    const [message, setMessage] = useState('');
    const [searchQuery, setSearchQuery] = useState('');
    const [filteredMyProjects, setFilteredMyProjects] = useState([]);
    const [filteredAvailableProjects, setFilteredAvailableProjects] = useState([]);

    useEffect(() => {
        if (user) {
            console.log('MyProjects: user changed, fetching projects:', user);
            fetchProjects();
        }
    }, [user]);

    useEffect(() => {
        // Фильтруем проекты по поисковому запросу
        const query = searchQuery.toLowerCase();
        
        const filtered_my = myProjects.filter(p => 
            p.name.toLowerCase().includes(query) || 
            (p.description && p.description.toLowerCase().includes(query)) ||
            (p.department && p.department.toLowerCase().includes(query))
        );
        
        const filtered_available = availableProjects.filter(p => 
            p.name.toLowerCase().includes(query) || 
            (p.description && p.description.toLowerCase().includes(query)) ||
            (p.department && p.department.toLowerCase().includes(query))
        );
        
        setFilteredMyProjects(filtered_my);
        setFilteredAvailableProjects(filtered_available);
    }, [searchQuery, myProjects, availableProjects]);

    const fetchProjects = async () => {
        try {
            setLoading(true);
            console.log('Fetching projects...');
            
            // Получить проекты пользователя
            const myRes = await api.get('/api/projects/user');
            console.log('My projects response:', myRes);
            const myProjectsList = Array.isArray(myRes.data) ? myRes.data : [];
            setMyProjects(myProjectsList);

            // Получить все доступные проекты
            const allRes = await api.get('/api/projects/available');
            console.log('All projects response:', allRes);
            const allProjectsList = Array.isArray(allRes.data) ? allRes.data : [];
            
            // Фильтруем доступные проекты - исключаем те что уже у пользователя
            const available = allProjectsList.filter(
                p => !myProjectsList.some(mp => mp.id === p.id)
            );
            console.log('Available projects after filter:', available);
            setAvailableProjects(available);
            
            if (myProjectsList.length === 0 && allProjectsList.length === 0) {
                showMessage('ℹ Нет проектов в системе', 'info');
            }
        } catch (error) {
            console.error('Error fetching projects:', error);
            const errorMsg = error.response?.data?.error || error.message || 'Неизвестная ошибка';
            showMessage('❌ Ошибка при загрузке проектов: ' + errorMsg, 'error');
        } finally {
            setLoading(false);
        }
    };

    const showMessage = (text, type = 'success') => {
        setMessage(text);
        setTimeout(() => setMessage(''), 3000);
    };

    const handleDragStart = (e, project, from) => {
        setDraggedProject({ project, from });
        e.dataTransfer.effectAllowed = 'move';
    };

    const handleDragOver = (e) => {
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';
    };

    const handleDrop = async (e, to) => {
        e.preventDefault();
        if (!draggedProject) return;

        const { project, from } = draggedProject;

        // Если перетаскивается из того же столбца - игнорируем
        if (from === to) {
            setDraggedProject(null);
            return;
        }

        try {
            setLoading(true);

            if (to === 'my') {
                // Добавить проект пользователю
                const response = await api.post('/api/projects/my/add', {
                    project_id: project.id
                });
                console.log('Add response:', response);
                showMessage(`✓ Проект "${project.name}" добавлен к вам`);
            } else if (to === 'available') {
                // Удалить проект у пользователя
                const response = await api.post('/api/projects/my/remove', {
                    project_id: project.id
                });
                console.log('Remove response:', response);
                showMessage(`✓ Проект "${project.name}" удален`);
            }

            // Обновляем список проектов после успешного изменения
            setTimeout(() => {
                fetchProjects();
            }, 500);
        } catch (error) {
            console.error('Error updating project:', error);
            const errorMsg = error.response?.data?.error || error.message || 'Неизвестная ошибка';
            showMessage('❌ Ошибка: ' + errorMsg, 'error');
        } finally {
            setLoading(false);
            setDraggedProject(null);
        }
    };

    return (
        <div className="my-projects-container">
            <h2>Мои проекты</h2>
            <p className="projects-description">
                Перетаскивайте проекты мышкой для добавления или удаления
            </p>

            {/* Поле поиска */}
            <div className="projects-search-container">
                <input
                    type="text"
                    placeholder="🔍 Поиск по названию, описанию или отделу..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="projects-search-input"
                    disabled={loading}
                />
                {searchQuery && (
                    <button
                        onClick={() => setSearchQuery('')}
                        className="projects-search-clear"
                        disabled={loading}
                    >
                        ✕
                    </button>
                )}
            </div>

            {message && (
                <div className={`message-banner ${message.includes('❌') || message.includes('Ошибка') ? 'error' : message.includes('ℹ') ? 'info' : 'success'}`}>
                    {message}
                </div>
            )}

            <div className="projects-drag-container">
                {/* Мои проекты */}
                <div className="projects-column">
                    <div className="column-header">
                        <h3>📌 Мои проекты ({filteredMyProjects.length})</h3>
                    </div>
                    <div
                        className="projects-list drag-zone"
                        onDragOver={handleDragOver}
                        onDrop={(e) => handleDrop(e, 'my')}
                        style={{
                            minHeight: '300px',
                            padding: '15px',
                            backgroundColor: 'rgba(76, 175, 80, 0.05)',
                            border: '2px dashed #4CAF50',
                            borderRadius: '8px'
                        }}
                    >
                        {filteredMyProjects.length === 0 ? (
                            <div className="empty-state">
                                <p>{searchQuery ? 'Нет проектов по этому запросу' : 'Нет назначенных проектов'}</p>
                                <p className="hint">{searchQuery ? 'Попробуйте другой поисковый запрос' : 'Перетащите проект сюда →'}</p>
                            </div>
                        ) : (
                            filteredMyProjects.map(project => (
                                <div
                                    key={project.id}
                                    draggable
                                    onDragStart={(e) => handleDragStart(e, project, 'my')}
                                    className="project-card draggable"
                                    style={{
                                        cursor: 'grab',
                                        opacity: draggedProject?.project.id === project.id ? 0.5 : 1
                                    }}
                                >
                                    <div className="project-card-title">{project.name}</div>
                                    <div className="project-card-desc">{project.description}</div>
                                    <div className="project-card-dept">📂 {project.department}</div>
                                    <div className="project-card-hint">Перетащить вправо для удаления ←</div>
                                </div>
                            ))
                        )}
                    </div>
                </div>

                {/* Доступные проекты */}
                <div className="projects-column">
                    <div className="column-header">
                        <h3>📂 Доступные проекты ({filteredAvailableProjects.length})</h3>
                    </div>
                    <div
                        className="projects-list drag-zone"
                        onDragOver={handleDragOver}
                        onDrop={(e) => handleDrop(e, 'available')}
                        style={{
                            minHeight: '300px',
                            padding: '15px',
                            backgroundColor: 'rgba(33, 150, 243, 0.05)',
                            border: '2px dashed #2196F3',
                            borderRadius: '8px'
                        }}
                    >
                        {filteredAvailableProjects.length === 0 ? (
                            <div className="empty-state">
                                <p>{searchQuery ? 'Нет проектов по этому запросу' : 'Нет доступных проектов'}</p>
                                <p className="hint">{searchQuery ? 'Попробуйте другой поисковый запрос' : 'Все проекты у вас в наличии ✓'}</p>
                            </div>
                        ) : (
                            filteredAvailableProjects.map(project => (
                                <div
                                    key={project.id}
                                    draggable
                                    onDragStart={(e) => handleDragStart(e, project, 'available')}
                                    className="project-card draggable available"
                                    style={{
                                        cursor: 'grab',
                                        opacity: draggedProject?.project.id === project.id ? 0.5 : 1
                                    }}
                                >
                                    <div className="project-card-title">{project.name}</div>
                                    <div className="project-card-desc">{project.description}</div>
                                    <div className="project-card-dept">📂 {project.department}</div>
                                    <div className="project-card-hint">Перетащить влево для добавления →</div>
                                </div>
                            ))
                        )}
                    </div>
                </div>
            </div>

            <div className="projects-footer">
                <button
                    className="btn btn-primary"
                    onClick={fetchProjects}
                    disabled={loading}
                >
                    🔄 Обновить
                </button>
                <p className="info-text">
                    💡 Совет: Перетаскивайте карточки проектов между столбцами для управления вашими проектами
                </p>
            </div>
        </div>
    );
};

export default MyProjects;
