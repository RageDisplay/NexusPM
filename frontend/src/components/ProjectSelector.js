import React, { useState, useEffect } from 'react';
import api from '../utils/api';

const ProjectSelector = ({ onProjectSelect, selectedProjectId, disabled = false, onlyActive = false, forceRefresh = 0 }) => {
    const [projects, setProjects] = useState([]);
    const [loading, setLoading] = useState(true);
    const [sortBy, setSortBy] = useState('priority'); // 'priority', 'date'
    const [searchQuery, setSearchQuery] = useState('');
    const [hideOutdated, setHideOutdated] = useState(true); // По умолчанию скрываем давно не обновлявшиеся

    useEffect(() => {
        fetchProjectsWithStats();
    }, [forceRefresh]);

    const fetchProjectsWithStats = async () => {
        try {
            setLoading(true);
            const response = await api.get('/api/projects/user/stats');
            console.log('Projects loaded from /api/projects/user/stats:', response.data);
            setProjects(response.data || []);
        } catch (error) {
            console.error('Error fetching projects from stats endpoint:', error);
            console.error('Error response:', error.response?.data);
            // Fallback: попробуем старый endpoint
            try {
                console.log('Falling back to /api/projects/user');
                const fallbackResponse = await api.get('/api/projects/user');
                console.log('Projects loaded from fallback:', fallbackResponse.data);
                let projectsWithStats = fallbackResponse.data || [];
                
                // Получаем last_report_date для каждого проекта если его нет
                projectsWithStats = await Promise.all(
                    projectsWithStats.map(async (project) => {
                        try {
                            const tasksResponse = await api.get('/api/tasks');
                            // Находим последний отчет для этого проекта
                            const projectTasks = tasksResponse.data.filter(t => t.project_id === project.id);
                            if (projectTasks.length > 0) {
                                const lastTask = projectTasks.reduce((prev, current) => 
                                    new Date(prev.created_at) > new Date(current.created_at) ? prev : current
                                );
                                return { ...project, last_report_date: lastTask.created_at };
                            }
                        } catch (e) {
                            console.error('Error fetching tasks for project', project.id, e);
                        }
                        return project;
                    })
                );
                
                setProjects(projectsWithStats);
            } catch (fallbackError) {
                console.error('Fallback also failed:', fallbackError);
                setProjects([]);
            }
        } finally {
            setLoading(false);
        }
    };

    // Вычисляем статус проекта на основе даты последнего отчета
    const getProjectStatus = (lastReportDate) => {
        if (!lastReportDate) {
            return { status: 'red', label: 'Давно не обновлялся' };
        }

        // Парсим дату правильно (убираем часовой пояс для локального времени)
        const lastDate = new Date(lastReportDate);
        const today = new Date();
        
        // Сбрасываем время для сравнения только по дате
        lastDate.setHours(0, 0, 0, 0);
        today.setHours(0, 0, 0, 0);

        // Вычисляем понедельник текущей недели
        const dayOfWeek = today.getDay();
        const monday = new Date(today);
        // Если воскресенье (0), то понедельник это дня 2 недели назад
        // Если другой день, то вычитаем дни с начала недели
        const daysToSubtract = dayOfWeek === 0 ? 6 : dayOfWeek - 1;
        monday.setDate(today.getDate() - daysToSubtract);
        monday.setHours(0, 0, 0, 0);

        const sunday = new Date(monday);
        sunday.setDate(monday.getDate() + 6);
        sunday.setHours(23, 59, 59, 999);

        // Проверяем если отчет за эту неделю
        if (lastDate >= monday && lastDate <= sunday) {
            return { status: 'green', label: 'Отчет за эту неделю' };
        }

        // Вычисляем разницу в днях
        const diffTime = today - lastDate;
        const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));

        if (diffDays <= 14) {
            return { status: 'yellow', label: `Просрочен на ${diffDays} дней` };
        }

        return { status: 'red', label: 'Давно не обновлялся' };
    };

    // Сортируем проекты
    const getSortedProjects = () => {
        let sorted = [...projects];

        // Поиск
        if (searchQuery) {
            sorted = sorted.filter(p =>
                p.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                p.description.toLowerCase().includes(searchQuery.toLowerCase())
            );
        }

        // Скрытие давно не обновлявшихся
        if (hideOutdated) {
            sorted = sorted.filter(p => {
                const status = getProjectStatus(p.last_report_date).status;
                return status !== 'red';
            });
        }

        // Сортировка
        if (sortBy === 'date') {
            // По дате последнего отчета (новые первыми)
            sorted.sort((a, b) => {
                const dateA = new Date(a.last_report_date || 0);
                const dateB = new Date(b.last_report_date || 0);
                return dateB - dateA;
            });
        } else {
            // По приоритету (зеленый > желтый > красный)
            const statusPriority = { green: 0, yellow: 1, red: 2 };
            sorted.sort((a, b) => {
                const statusA = getProjectStatus(a.last_report_date).status;
                const statusB = getProjectStatus(b.last_report_date).status;
                return statusPriority[statusA] - statusPriority[statusB];
            });
        }

        return sorted;
    };

    const sortedProjects = getSortedProjects();
    const selectedProject = projects.find(p => p.id === selectedProjectId);

    return (
        <div className="project-selector">
            <div className="project-selector-header">
                <div className="search-and-sort">
                    <input
                        type="text"
                        placeholder="🔍 Поиск проектов..."
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                        className="project-search"
                        disabled={disabled || loading}
                    />
                    <div className="sort-controls">
                        <label className="hide-outdated-checkbox">
                            <input 
                                type="checkbox" 
                                checked={hideOutdated}
                                onChange={(e) => setHideOutdated(e.target.checked)}
                                disabled={disabled || loading}
                            />
                            Скрыть давно не обновлявшиеся
                        </label>
                        <select
                            value={sortBy}
                            onChange={(e) => setSortBy(e.target.value)}
                            className="sort-select"
                            disabled={disabled || loading}
                        >
                            <option value="priority">По приоритету</option>
                            <option value="date">По дате отчета</option>
                        </select>
                    </div>
                </div>
            </div>

            {loading ? (
                <div className="project-selector-loading">
                    <span>⏳ Загрузка проектов...</span>
                </div>
            ) : (
                <>
                    <div className="project-cards-container">
                        {sortedProjects.length > 0 ? (
                            sortedProjects.map(project => {
                                const projectStatus = getProjectStatus(project.last_report_date);
                                const isSelected = selectedProjectId === project.id;
                                const lastReportText = project.last_report_date
                                    ? new Date(project.last_report_date).toLocaleDateString('ru-RU')
                                    : 'Нет отчетов';

                                return (
                                    <div
                                        key={project.id}
                                        className={`project-card status-${projectStatus.status} ${isSelected ? 'selected' : ''}`}
                                        onClick={() => !disabled && onProjectSelect(project)}
                                        role="button"
                                        tabIndex={disabled ? -1 : 0}
                                        onKeyPress={(e) => {
                                            if (e.key === 'Enter' && !disabled) {
                                                onProjectSelect(project);
                                            }
                                        }}
                                    >
                                        <div className="project-card-header">
                                            <h4>{project.name}</h4>
                                            <div className={`status-indicator status-${projectStatus.status}`} title={projectStatus.label}></div>
                                        </div>

                                        <p className="project-description">{project.description || 'Без описания'}</p>

                                        <div className="project-card-footer">
                                            <div className="last-report-info">
                                                <span className="label">Последний отчет:</span>
                                                <span className="date">{lastReportText}</span>
                                            </div>
                                            <div className={`status-label status-${projectStatus.status}`}>
                                                {projectStatus.label}
                                            </div>
                                        </div>

                                        {isSelected && (
                                            <div className="selection-checkmark">✓</div>
                                        )}
                                    </div>
                                );
                            })
                        ) : (
                            <div className="empty-projects">
                                <p>Проектов не найдено {searchQuery ? 'по вашему поиску' : 'для вас'}</p>
                            </div>
                        )}
                    </div>

                    {selectedProject && (
                        <div className="selected-project-info">
                            <strong>Выбран проект:</strong> {selectedProject.name}
                        </div>
                    )}
                </>
            )}
        </div>
    );
};

export default ProjectSelector;
