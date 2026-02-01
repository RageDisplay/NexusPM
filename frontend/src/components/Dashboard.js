import React, { useState, useEffect } from 'react';
import api from '../utils/api';
import { useAuth } from '../contexts/AuthContext';
import DepartmentStatistics from './DepartmentStatistics';
import ActivityHistory from './ActivityHistory';
import Notifications from './Notifications';
import UserProfile from './UserProfile';
import MyProjects from './MyProjects';
import { Line, Bar } from 'react-chartjs-2';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Title,
  Tooltip,
  Legend
} from 'chart.js';

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Title,
  Tooltip,
  Legend
);

const Dashboard = () => {
  const { user } = useAuth();
  const [statistics, setStatistics] = useState(null);
  const [weeklyLoad, setWeeklyLoad] = useState([]);
  const [progressDynamics, setProgressDynamics] = useState([]);
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState('overview');

  useEffect(() => {
    if (user) {
      fetchDashboardData();
    }
  }, [user]);

  const fetchDashboardData = async () => {
    try {
      setLoading(true);
      const [statsRes, loadRes, progressRes] = await Promise.all([
        api.get('/api/dashboard/statistics'),
        api.get('/api/dashboard/weekly-load'),
        api.get('/api/dashboard/progress-dynamics')
      ]);
      
      setStatistics(statsRes.data);
      setWeeklyLoad(loadRes.data || []);
      setProgressDynamics(progressRes.data || []);
    } catch (error) {
      console.error('Error fetching dashboard data:', error);
    } finally {
      setLoading(false);
    }
  };

  const weeklyLoadChartData = {
    labels: weeklyLoad.map(w => `Неделя ${w.week}`),
    datasets: [
      {
        label: 'Часы в неделю',
        data: weeklyLoad.map(w => w.load_hours),
        backgroundColor: 'rgba(75, 192, 192, 0.5)',
        borderColor: 'rgba(75, 192, 192, 1)',
        borderWidth: 2
      }
    ]
  };

  const progressChartData = {
    labels: progressDynamics.map(p => new Date(p.date).toLocaleDateString('ru-RU')),
    datasets: [
      {
        label: 'Средний прогресс (%)',
        data: progressDynamics.map(p => p.progress),
        borderColor: 'rgba(153, 102, 255, 1)',
        backgroundColor: 'rgba(153, 102, 255, 0.1)',
        tension: 0.3,
        fill: true
      }
    ]
  };

  const chartOptions = {
    responsive: true,
    plugins: {
      legend: {
        position: 'top',
      },
      title: {
        display: true,
        font: {
          size: 14
        }
      }
    },
    scales: {
      y: {
        beginAtZero: true
      }
    }
  };

  return (
    <div className="dashboard-container">
      <h1>Дашборд</h1>
      <p className="welcome-text">Привет, {user?.username}!</p>

      <div className="dashboard-tabs">
        <button 
          className={`tab-button ${activeTab === 'overview' ? 'active' : ''}`}
          onClick={() => setActiveTab('overview')}
        >
          Обзор
        </button>
        <button 
          className={`tab-button ${activeTab === 'projects' ? 'active' : ''}`}
          onClick={() => setActiveTab('projects')}
        >
          Мои проекты
        </button>
        <button 
          className={`tab-button ${activeTab === 'profile' ? 'active' : ''}`}
          onClick={() => setActiveTab('profile')}
        >
          Профиль
        </button>
        <button 
          className={`tab-button ${activeTab === 'activity' ? 'active' : ''}`}
          onClick={() => setActiveTab('activity')}
        >
          Активность
        </button>
        <button 
          className={`tab-button ${activeTab === 'statistics' ? 'active' : ''}`}
          onClick={() => setActiveTab('statistics')}
        >
          Статистика
        </button>
      </div>

      {activeTab === 'overview' && (
        <>
          <div className="dashboard-grid">
            {/* Статистика по задачам */}
            {statistics && (
              <>
                <div className="dashboard-card">
                  <h3>Мои задачи <span className="info-badge">за 7 дней</span></h3>
                  <div className="card-content">
                    <div className="stat-row">
                      <span>Всего задач:</span>
                      <strong>{statistics.total_tasks}</strong>
                    </div>
                    <div className="stat-row">
                      <span>Активных:</span>
                      <strong className="active">{statistics.active_tasks}</strong>
                    </div>
                    <div className="stat-row">
                      <span>Завершено:</span>
                      <strong className="completed">{statistics.completed_tasks}</strong>
                    </div>
                    <div className="stat-row">
                      <span>Средний прогресс:</span>
                      <strong>{Math.round(statistics.average_progress)}%</strong>
                    </div>
                  </div>
                </div>

                <div className="dashboard-card">
                  <h3>Загруженность <span className="info-badge">за 7 дней</span></h3>
                  <div className="card-content">
                    <div className="stat-row">
                      <span>Часов в неделю:</span>
                      <strong>{statistics.total_hours_per_week.toFixed(1)}</strong>
                    </div>
                    <div className="stat-row">
                      <span>Загрузка в месяц:</span>
                      <strong>{statistics.total_load_per_month}%</strong>
                    </div>
                  </div>
                </div>
              </>
            )}

            {/* Роль и Отдел */}
            <div className="dashboard-card">
              <h3>Информация</h3>
              <div className="card-content">
                <div className="stat-row">
                  <span>Роль:</span>
                  <strong>
                    {user?.role === 'user' ? 'Сотрудник' : 
                     user?.role === 'manager' ? 'Менеджер' : 
                     user?.role === 'admin' ? 'Администратор' : user?.role}
                  </strong>
                </div>
                <div className="stat-row">
                  <span>Отдел:</span>
                  <strong>{user?.department || 'Не установлен'}</strong>
                </div>
              </div>
            </div>
          </div>

          {/* Графики */}
          {weeklyLoad.length > 0 && (
            <div className="dashboard-chart">
              <h3>График загруженности (текущий месяц)</h3>
              <Bar data={weeklyLoadChartData} options={chartOptions} />
            </div>
          )}

          {progressDynamics.length > 0 && (
            <div className="dashboard-chart">
              <h3>Динамика прогресса (последние 30 дней)</h3>
              <Line data={progressChartData} options={chartOptions} />
            </div>
          )}

          {/* Уведомления */}
          <div className="dashboard-notifications">
            <Notifications />
          </div>
        </>
      )}

      {activeTab === 'projects' && (
        <MyProjects />
      )}

      {activeTab === 'profile' && (
        <UserProfile />
      )}

      {activeTab === 'activity' && (
        <ActivityHistory />
      )}

      {activeTab === 'statistics' && (
        <DepartmentStatistics />
      )}
    </div>
  );
};

export default Dashboard;