import React from 'react';
import api from '../utils/api'; 
import { useAuth } from '../contexts/AuthContext';

const Reports = () => {
    const { user } = useAuth();

    const downloadReport = async (type) => {
        try {
            const response = await api.get(`/api/reports/${type}`, { 
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