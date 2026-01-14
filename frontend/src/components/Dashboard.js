import React from 'react';
import { useAuth } from '../contexts/AuthContext';
import DepartmentStatistics from './DepartmentStatistics';

const Dashboard = () => {
  const { user } = useAuth();

  return (
    <div>
      <h1>Дешборд</h1>
      <p>Привет, {user?.username}!</p>
      <div className="dashboard">
        <div className="dashboard-card">
          <h3>Ваша роль в системе</h3>
          <div className="number">{user?.role}</div>
        </div>
        <div className="dashboard-card">
          <h3>Отдел</h3>
          <div className="number">{user?.department || 'Не установлен'}</div>
        </div>
      </div>
      
      {(user?.role === 'manager' || user?.role === 'admin') && (
        <DepartmentStatistics />
      )}
    </div>
  );
};

export default Dashboard;