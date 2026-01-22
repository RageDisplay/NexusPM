import React from 'react';
import api from '../utils/api'; 
import { useAuth } from '../contexts/AuthContext';

const Reports = () => {
    const { user } = useAuth();
    const [startDate, setStartDate] = React.useState('');
    const [endDate, setEndDate] = React.useState('');

    React.useEffect(() => {
        // Устанавливаем начало недели (понедельник) и конец (воскресенье)
        const today = new Date();
        const dayOfWeek = today.getDay();
        const diff = today.getDate() - dayOfWeek + (dayOfWeek === 0 ? -6 : 1);
        
        const monday = new Date(today.setDate(diff));
        const sunday = new Date(monday);
        sunday.setDate(sunday.getDate() + 6);
        
        const formatDate = (date) => date.toISOString().split('T')[0];
        setStartDate(formatDate(monday));
        setEndDate(formatDate(sunday));
    }, []);

    const downloadReport = async (type) => {
        try {
            const params = {};
            if (startDate && endDate) {
                params.start_date = startDate;
                params.end_date = endDate;
            }
            
            const response = await api.get(`/api/reports/${type}`, { 
                params,
                responseType: 'blob'
            });

      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;
      
      let filename = '';
      switch (type) {
        case 'my-tasks':
          filename = 'my_tasks.xlsx';
          break;
        case 'department-tasks':
          filename = 'department_statistics.xlsx';
          break;
        case 'all-tasks':
          filename = 'statistics_by_projects.xlsx';
          break;
        default:
          filename = 'report.xlsx';
      }
      
      link.setAttribute('download', filename);
      document.body.appendChild(link);
      link.click();
      link.remove();
    } catch (error) {
            console.error('Ошибка в скачивании отчёта:', error);
            alert('Ошибка в скачивании отчёта: ' + (error.response?.data?.error || error.message));
        }
    };

  return (
    <div className="reports-section">
      <h2>Отчёты</h2>
      <p>Скачать отчёты в формате excel.</p>
      
      <div className="report-date-range">
        <h3>Фильтр по датам:</h3>
        <div>
          <div>
            <label htmlFor="report-start-date">С:</label>
            <input
              id="report-start-date"
              type="date"
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
            />
          </div>
          <div>
            <label htmlFor="report-end-date">По:</label>
            <input
              id="report-end-date"
              type="date"
              value={endDate}
              onChange={(e) => setEndDate(e.target.value)}
            />
          </div>
        </div>
      </div>
      
      <div className="report-options">
        <button 
          className="btn btn-primary"
          onClick={() => downloadReport('my-tasks')}
        >
          Скачать мои задачи
        </button>

        {(user.role === 'manager' || user.role === 'admin') && (
          <button 
            className="btn btn-secondary"
            onClick={() => downloadReport('department-tasks')}
          >
            Скачать статистику отдела
          </button>
        )}

        {user.role === 'admin' && (
          <button 
            className="btn btn-secondary"
            onClick={() => downloadReport('all-tasks')}
          >
            Скачать статистику по проектам
          </button>
        )}
      </div>
    </div>
  );
};

export default Reports;