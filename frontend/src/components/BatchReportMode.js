import React, { useState, useEffect } from 'react';
import api from '../utils/api';
import { useAuth } from '../contexts/AuthContext';

const BatchReportMode = () => {
    const { user } = useAuth();
    const [mode, setMode] = useState('select'); // 'select', 'fill', 'review', 'done'
    const [reports, setReports] = useState([]); // Массив заполненных отчетов
    const [currentProjectIndex, setCurrentProjectIndex] = useState(0);
    const [projects, setProjects] = useState([]);
    const [selectedProjectIds, setSelectedProjectIds] = useState(new Set());
    const [hideOutdated, setHideOutdated] = useState(true); // Фильтр для скрытия старых проектов (по умолчанию включен)
    const [currentForm, setCurrentForm] = useState({
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
    const [message, setMessage] = useState(null);
    const [lastTaskData, setLastTaskData] = useState(null); // Информация о последнем отчёте текущего проекта
    const [loadingLastTask, setLoadingLastTask] = useState(false);

    useEffect(() => {
        fetchProjects();
    }, []);

    const fetchProjects = async () => {
        try {
            const response = await api.get('/api/projects/user/stats');
            console.log('Projects from /api/projects/user/stats:', response.data);
            setProjects(response.data || []);
        } catch (error) {
            console.error('Error fetching projects from /api/projects/user/stats:', error);
            // Fallback на /api/projects/user если stats эндпоинт недоступен
            try {
                console.log('Trying fallback: /api/projects/user');
                const fallbackResponse = await api.get('/api/projects/user');
                console.log('Projects from /api/projects/user:', fallbackResponse.data);
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
                console.error('Error fetching projects from fallback:', fallbackError);
                setProjects([]);
            }
        }
    };

    const handleProjectSelectForBatch = (project) => {
        const newSelected = new Set(selectedProjectIds);
        if (newSelected.has(project.id)) {
            newSelected.delete(project.id);
        } else {
            newSelected.add(project.id);
        }
        setSelectedProjectIds(newSelected);
    };

    // Вычисляем статус проекта на основе даты последнего отчета
    const getProjectStatus = (lastReportDate) => {
        if (!lastReportDate) {
            return { status: 'red', label: 'Нет отчетов' };
        }

        // Парсим дату правильно (убираем часовой пояс для локального времени)
        const lastDate = new Date(lastReportDate);
        const today = new Date();
        
        // Сбрасываем время для сравнения только по дате
        lastDate.setHours(0, 0, 0, 0);
        today.setHours(0, 0, 0, 0);

        console.log('[Status Debug]', {
            lastReportDate: lastReportDate,
            parsedDate: lastDate.toISOString(),
            todayDate: today.toISOString(),
            dayOfWeek: today.getDay(),
            diffDays: Math.ceil((today - lastDate) / (1000 * 60 * 60 * 24))
        });

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

        console.log('[Week Range]', {
            monday: monday.toLocaleDateString('ru-RU'),
            sunday: sunday.toLocaleDateString('ru-RU'),
            lastDateLocal: lastDate.toLocaleDateString('ru-RU')
        });

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

    const fetchLastTask = async (projectId) => {
        try {
            setLoadingLastTask(true);
            const response = await api.get(`/api/tasks/project/${projectId}/last`);
            setLastTaskData(response.data);
        } catch (error) {
            // Нет предыдущих задач - это нормально
            setLastTaskData(null);
        } finally {
            setLoadingLastTask(false);
        }
    };

    // Фильтруем и сортируем проекты
    const getDisplayedProjects = () => {
        let displayed = [...projects];

        // Фильтруем устаревшие если нужно
        if (hideOutdated) {
            displayed = displayed.filter(p => {
                const status = getProjectStatus(p.last_report_date);
                return status.status !== 'red';
            });
        }

        // Сортируем по статусу (зеленые->желтые->красные)
        displayed.sort((a, b) => {
            const statusA = getProjectStatus(a.last_report_date).status;
            const statusB = getProjectStatus(b.last_report_date).status;
            const statusOrder = { 'green': 0, 'yellow': 1, 'red': 2 };
            return statusOrder[statusA] - statusOrder[statusB];
        });

        return displayed;
    };

    const startFilling = () => {
        if (selectedProjectIds.size === 0) {
            alert('Выберите хотя бы один проект');
            return;
        }
        setReports([]);
        setCurrentProjectIndex(0);
        setMode('fill');
        initializeFormForProject(0);
    };

    const initializeFormForProject = (index) => {
        const selectedIds = Array.from(selectedProjectIds);
        if (index < selectedIds.length) {
            const projectId = selectedIds[index];
            const project = projects.find(p => p.id === projectId);
            setCurrentForm({
                title: project?.name || '',
                description: project?.description || '',
                progress: 0,
                hours_per_week: 0,
                load_per_month: 0,
                project_id: projectId,
                user_id: user?.id || 0,
                weekly_info: '',
                planning: '',
                help_needed: ''
            });
            // Загружаем последний отчёт для этого проекта
            fetchLastTask(projectId);
        }
    };

    const getCurrentProject = () => {
        const selectedIds = Array.from(selectedProjectIds);
        return projects.find(p => p.id === selectedIds[currentProjectIndex]);
    };

    const saveCurrentReport = () => {
        const reportToSave = { ...currentForm };
        const newReports = [...reports, reportToSave];
        setReports(newReports);

        // Переходим к следующему проекту
        const selectedIds = Array.from(selectedProjectIds);
        if (currentProjectIndex + 1 < selectedIds.length) {
            setCurrentProjectIndex(currentProjectIndex + 1);
            initializeFormForProject(currentProjectIndex + 1);
            setMessage({ type: 'success', text: 'Отчет сохранен. Переходим к следующему проекту.' });
            setTimeout(() => setMessage(null), 3000);
        } else {
            // Все отчеты заполнены - переходим на review
            setMode('review');
            setLastTaskData(null);
        }
    };

    const skipCurrentProject = () => {
        const selectedIds = Array.from(selectedProjectIds);
        if (currentProjectIndex + 1 < selectedIds.length) {
            setCurrentProjectIndex(currentProjectIndex + 1);
            initializeFormForProject(currentProjectIndex + 1);
        } else {
            setMode('review');
            setLastTaskData(null);
        }
    };

    const submitAllReports = async () => {
        if (reports.length === 0) {
            alert('Нет отчетов для отправки');
            return;
        }

        try {
            setLoading(true);
            const results = [];
            let successCount = 0;
            let errorCount = 0;

            for (const report of reports) {
                try {
                    const response = await api.post('/api/tasks', report);
                    results.push({ success: true, data: response.data });
                    successCount++;
                } catch (error) {
                    results.push({ success: false, error: error.message });
                    errorCount++;
                }
            }

            setMode('done');
            setMessage({
                type: 'success',
                text: `Загружено: ${successCount}, Ошибок: ${errorCount}`
            });

            // Очистка после успеха
            setTimeout(() => {
                setMode('select');
                setReports([]);
                setSelectedProjectIds(new Set());
                setCurrentProjectIndex(0);
                setMessage(null);
                fetchProjects();
            }, 2000);
        } catch (error) {
            console.error('Error submitting reports:', error);
            setMessage({ type: 'error', text: 'Ошибка при отправке отчетов' });
        } finally {
            setLoading(false);
        }
    };

    const editReport = (index) => {
        setCurrentForm(reports[index]);
        setCurrentProjectIndex(index);
        setMode('fill');
    };

    const deleteReport = (index) => {
        setReports(reports.filter((_, i) => i !== index));
    };

    return (
        <div className="batch-report-mode">
            {message && (
                <div className={`message message-${message.type}`}>
                    {message.text}
                </div>
            )}

            {/* Режим выбора проектов */}
            {mode === 'select' && (
                <div className="batch-select-section">
                    <div className="batch-select-header">
                        <h3>📋 Массовое заполнение отчетов</h3>
                        <span className="batch-selected-count">{selectedProjectIds.size} выбрано</span>
                    </div>
                    <p style={{ color: 'var(--text-secondary)', marginBottom: '15px' }}>
                        Выберите проекты, по которым нужны отчеты. Затем заполните их один за другим и отправьте все разом.
                    </p>

                    {/* Фильтр для скрытия давно не обновлявшихся */}
                    <label className="hide-outdated-checkbox" style={{ marginBottom: '20px' }}>
                        <input 
                            type="checkbox" 
                            checked={hideOutdated} 
                            onChange={(e) => setHideOutdated(e.target.checked)}
                        />
                        <span>Скрыть давно не обновлявшиеся</span>
                    </label>

                    <div className="project-cards-container" style={{ marginBottom: '20px', maxHeight: '400px', overflowY: 'auto' }}>
                        {getDisplayedProjects().length === 0 ? (
                            <p style={{ color: 'var(--text-secondary)', gridColumn: '1 / -1' }}>
                                {hideOutdated ? 'Все проекты давно не обновлялись' : 'Нет доступных проектов'}
                            </p>
                        ) : (
                            getDisplayedProjects().map(project => {
                                const projectStatus = getProjectStatus(project.last_report_date);
                                return (
                                    <div
                                        key={project.id}
                                        className={`project-card status-${projectStatus.status} ${selectedProjectIds.has(project.id) ? 'selected' : ''}`}
                                        onClick={() => handleProjectSelectForBatch(project)}
                                        style={{ cursor: 'pointer', border: selectedProjectIds.has(project.id) ? '2px solid var(--primary)' : '2px solid rgba(255, 159, 67, 0.15)' }}
                                    >
                                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '8px' }}>
                                            <h4 style={{ margin: '0', color: 'var(--text-primary)', flex: 1 }}>
                                                {selectedProjectIds.has(project.id) && '✓ '}
                                                {project.name}
                                            </h4>
                                            <div 
                                                className={`status-indicator status-${projectStatus.status}`}
                                                style={{ 
                                                    width: '12px', 
                                                    height: '12px', 
                                                    borderRadius: '50%',
                                                    flexShrink: 0,
                                                    marginLeft: '8px',
                                                    backgroundColor: projectStatus.status === 'green' ? '#4caf50' : projectStatus.status === 'yellow' ? '#ffc107' : '#f44336'
                                                }}
                                            />
                                        </div>
                                        <p style={{ margin: '0 0 8px 0', color: 'var(--text-secondary)', fontSize: '0.75rem' }}>
                                            {projectStatus.label}
                                        </p>
                                        <p style={{ margin: '0 0 8px 0', color: 'var(--text-secondary)', fontSize: '0.85rem', lineHeight: '1.4' }}>
                                            {project.description}
                                        </p>
                                        {project.last_report_date && (
                                            <p style={{ margin: 0, color: 'var(--text-secondary)', fontSize: '0.75rem' }}>
                                                Последний отчет: {new Date(project.last_report_date).toLocaleDateString('ru-RU')}
                                            </p>
                                        )}
                                    </div>
                                );
                            })
                        )}
                    </div>

                    <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end' }}>
                        <button
                            className="btn btn-secondary"
                            onClick={() => setSelectedProjectIds(new Set())}
                            disabled={selectedProjectIds.size === 0}
                        >
                            Очистить выбор
                        </button>
                        <button
                            className="btn btn-primary"
                            onClick={startFilling}
                            disabled={selectedProjectIds.size === 0 || loading}
                        >
                            Начать заполнение ({selectedProjectIds.size})
                        </button>
                    </div>
                </div>
            )}

            {/* Режим заполнения отчета */}
            {mode === 'fill' && (
                <div className="batch-fill-section">
                    <div className="fill-header">
                        <h3>Проект {currentProjectIndex + 1} из {selectedProjectIds.size}</h3>
                        <span className="project-name">{getCurrentProject()?.name}</span>
                        <div className="progress-bar">
                            <div
                                className="progress-fill"
                                style={{ width: `${((currentProjectIndex + 1) / selectedProjectIds.size) * 100}%` }}
                            ></div>
                        </div>
                    </div>

                    {/* Информация о последнем отчете */}
                    {lastTaskData && (
                        <div style={{
                            backgroundColor: 'rgba(64, 224, 208, 0.08)',
                            border: '2px solid rgba(64, 224, 208, 0.3)',
                            borderLeft: '4px solid rgba(64, 224, 208, 0.6)',
                            borderRadius: '8px',
                            padding: '15px',
                            marginBottom: '20px'
                        }}>
                            <p style={{ margin: '0 0 12px 0', fontSize: '0.95rem', fontWeight: '600', color: 'var(--text-primary)' }}>
                                Ваша последняя запись по этому проекту:
                            </p>
                            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px', marginBottom: '12px' }}>
                                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '8px 0', borderBottom: '1px solid rgba(100, 181, 246, 0.1)' }}>
                                    <span style={{ color: 'var(--text-secondary)', fontSize: '0.85rem' }}>
                                        <strong>Прогресс:</strong>
                                    </span>
                                    <span style={{ color: 'var(--text-primary)', fontSize: '0.9rem', fontWeight: '500' }}>
                                        {lastTaskData.progress}%
                                    </span>
                                </div>
                                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '8px 0', borderBottom: '1px solid rgba(100, 181, 246, 0.1)' }}>
                                    <span style={{ color: 'var(--text-secondary)', fontSize: '0.85rem' }}>
                                        <strong>Часов потрачено:</strong>
                                    </span>
                                    <span style={{ color: 'var(--text-primary)', fontSize: '0.9rem', fontWeight: '500' }}>
                                        {lastTaskData.hours_per_week} ч
                                    </span>
                                </div>
                                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '8px 0', borderBottom: '1px solid rgba(100, 181, 246, 0.1)' }}>
                                    <span style={{ color: 'var(--text-secondary)', fontSize: '0.85rem' }}>
                                        <strong>Загрузка на месяц:</strong>
                                    </span>
                                    <span style={{ color: 'var(--text-primary)', fontSize: '0.9rem', fontWeight: '500' }}>
                                        {lastTaskData.load_per_month}%
                                    </span>
                                </div>
                            </div>
                            
                            {lastTaskData.weekly_info && (
                                <div style={{ marginBottom: '10px', paddingBottom: '10px', borderBottom: '1px solid rgba(100, 181, 246, 0.1)' }}>
                                    <p style={{ margin: '0 0 5px 0', fontSize: '0.85rem', fontWeight: '500', color: 'var(--text-secondary)' }}>
                                        За неделю:
                                    </p>
                                    <p style={{ margin: '0', fontSize: '0.85rem', color: 'var(--text-primary)', lineHeight: '1.4', wordWrap: 'break-word', overflowWrap: 'break-word', whiteSpace: 'pre-wrap' }}>
                                        {lastTaskData.weekly_info}
                                    </p>
                                </div>
                            )}
                            
                            {lastTaskData.planning && (
                                <div style={{ marginBottom: '10px', paddingBottom: '10px', borderBottom: '1px solid rgba(100, 181, 246, 0.1)' }}>
                                    <p style={{ margin: '0 0 5px 0', fontSize: '0.85rem', fontWeight: '500', color: 'var(--text-secondary)' }}>
                                        Планируется:
                                    </p>
                                    <p style={{ margin: '0', fontSize: '0.85rem', color: 'var(--text-primary)', lineHeight: '1.4', wordWrap: 'break-word', overflowWrap: 'break-word', whiteSpace: 'pre-wrap' }}>
                                        {lastTaskData.planning}
                                    </p>
                                </div>
                            )}
                            
                            {lastTaskData.help_needed && (
                                <div style={{ marginBottom: '10px', paddingBottom: '10px', borderBottom: '1px solid rgba(100, 181, 246, 0.1)' }}>
                                    <p style={{ margin: '0 0 5px 0', fontSize: '0.85rem', fontWeight: '500', color: 'var(--text-secondary)' }}>
                                        Требуется помощь:
                                    </p>
                                    <p style={{ margin: '0', fontSize: '0.85rem', color: 'var(--text-primary)', lineHeight: '1.4', wordWrap: 'break-word', overflowWrap: 'break-word', whiteSpace: 'pre-wrap' }}>
                                        {lastTaskData.help_needed}
                                    </p>
                                </div>
                            )}
                            
                            <p style={{ margin: '0', fontSize: '0.8rem', color: 'var(--text-secondary)', textAlign: 'right' }}>
                                Обновлено: {new Date(lastTaskData.updated_at).toLocaleDateString('ru-RU')}
                            </p>
                        </div>
                    )}

                    {loadingLastTask && (
                        <div style={{
                            backgroundColor: 'rgba(100, 181, 246, 0.08)',
                            border: '1px solid rgba(100, 181, 246, 0.2)',
                            borderRadius: '8px',
                            padding: '12px 15px',
                            marginBottom: '20px',
                            textAlign: 'center',
                            color: 'var(--text-secondary)',
                            fontSize: '0.85rem'
                        }}>
                            ⏳ Загрузка предыдущей информации...
                        </div>
                    )}

                    <div className="batch-form">
                        <div className="form-group">
                            <label>Информация за неделю</label>
                            <textarea
                                placeholder="Что было сделано за неделю"
                                value={currentForm.weekly_info}
                                onChange={(e) => setCurrentForm({ ...currentForm, weekly_info: e.target.value })}
                                rows="4"
                            />
                        </div>

                        <div className="form-group">
                            <label>Планирование на следующую неделю</label>
                            <textarea
                                placeholder="Что планируется на следующую неделю"
                                value={currentForm.planning}
                                onChange={(e) => setCurrentForm({ ...currentForm, planning: e.target.value })}
                                rows="4"
                            />
                        </div>

                        <div className="form-row">
                            <div className="form-group">
                                <label>Прогресс: {currentForm.progress}%</label>
                                <input
                                    type="range"
                                    min="0"
                                    max="100"
                                    value={currentForm.progress}
                                    onChange={(e) => setCurrentForm({ ...currentForm, progress: parseInt(e.target.value) })}
                                />
                            </div>

                            <div className="form-group">
                                <label>Часов потрачено</label>
                                <input
                                    type="number"
                                    step="0.5"
                                    min="0"
                                    value={currentForm.hours_per_week}
                                    onChange={(e) => setCurrentForm({ ...currentForm, hours_per_week: parseFloat(e.target.value) || 0 })}
                                />
                            </div>

                            <div className="form-group">
                                <label>Загрузка на месяц (%)</label>
                                <input
                                    type="number"
                                    min="0"
                                    max="100"
                                    value={currentForm.load_per_month}
                                    onChange={(e) => setCurrentForm({ ...currentForm, load_per_month: parseInt(e.target.value) || 0 })}
                                />
                            </div>
                        </div>

                        <div className="form-group">
                            <label>Требуется помощь</label>
                            <input
                                type="text"
                                placeholder="Укажите помощь, если требуется"
                                value={currentForm.help_needed}
                                onChange={(e) => setCurrentForm({ ...currentForm, help_needed: e.target.value })}
                            />
                        </div>
                    </div>

                    <div className="batch-actions">
                        <button
                            className="btn btn-primary"
                            onClick={saveCurrentReport}
                            disabled={loading}
                        >
                            ✓ Сохранить и далее
                        </button>
                        <button
                            className="btn btn-secondary"
                            onClick={skipCurrentProject}
                            disabled={loading}
                        >
                            ⊘ Пропустить проект
                        </button>
                        <button
                            className="btn btn-danger"
                            onClick={() => {
                                setMode('select');
                                setLastTaskData(null);
                            }}
                            disabled={loading}
                        >
                            ✕ Отмена
                        </button>
                    </div>
                </div>
            )}

            {/* Режим проверки */}
            {mode === 'review' && (
                <div className="batch-review-section">
                    <h3>Проверка отчетов ({reports.length})</h3>

                    <div className="reports-list">
                        {reports.map((report, idx) => {
                            const project = projects.find(p => p.id === report.project_id);
                            return (
                                <div key={idx} className="report-item">
                                    <div className="report-header">
                                        <h4>{project?.name}</h4>
                                        <div className="report-stats">
                                            <span>Прогресс: {report.progress}%</span>
                                            <span>Часов: {report.hours_per_week}ч</span>
                                            <span>Загрузка: {report.load_per_month}%</span>
                                        </div>
                                    </div>
                                    <p><strong>За неделю:</strong> {report.weekly_info || 'Не заполнено'}</p>
                                    <p><strong>Планирование:</strong> {report.planning || 'Не заполнено'}</p>
                                    {report.help_needed && <p><strong>Помощь:</strong> {report.help_needed}</p>}
                                    <div className="report-actions">
                                        <button className="btn btn-secondary" onClick={() => editReport(idx)}>
                                            Редактировать
                                        </button>
                                        <button className="btn btn-danger" onClick={() => deleteReport(idx)}>
                                            ✕ Удалить
                                        </button>
                                    </div>
                                </div>
                            );
                        })}
                    </div>

                    <div className="batch-actions">
                        <button
                            className="btn btn-primary"
                            onClick={submitAllReports}
                            disabled={loading || reports.length === 0}
                        >
                            ✓ Отправить все отчеты ({reports.length})
                        </button>
                        <button
                            className="btn btn-secondary"
                            onClick={() => setMode('select')}
                            disabled={loading}
                        >
                            + Добавить еще
                        </button>
                        <button
                            className="btn btn-danger"
                            onClick={() => {
                                setMode('select');
                                setReports([]);
                            }}
                            disabled={loading}
                        >
                            ✕ Отмена
                        </button>
                    </div>
                </div>
            )}

            {/* Режим завершения */}
            {mode === 'done' && (
                <div className="batch-done-section">
                    <h3>✓ Отчеты успешно загружены!</h3>
                    <p>Все отчеты добавлены в базу данных.</p>
                </div>
            )}
        </div>
    );
};

export default BatchReportMode;
