import React from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import Login from './components/Login';
import Dashboard from './components/Dashboard';
import TaskManager from './components/TaskManager';
import UserManagement from './components/UserManagement';
import ProjectManagement from './components/ProjectManagement';
import Reports from './components/Reports';
import Navigation from './components/Navigation';
import BackupManager from './components/BackupManager';
import './App.css';

function ProtectedRoute({ children, requiredRole }) {
  const { user, loading } = useAuth();
  
  if (loading) return <div>Загрузка...</div>;
  if (!user) return <Navigate to="/login" />;
  
  // Если требуется несколько ролей, передаём массив
  if (requiredRole) {
    const roles = Array.isArray(requiredRole) ? requiredRole : [requiredRole];
    if (!roles.includes(user.role) && user.role !== 'admin') {
      return <Navigate to="/dashboard" />;
    }
  }
  
  return children;
}

function App() {
  return (
    <AuthProvider>
      <Router>
        <div className="App">
          <Navigation />
          <div className="container">
            <Routes>
              <Route path="/login" element={<Login />} />
              <Route path="/dashboard" element={
                <ProtectedRoute>
                  <Dashboard />
                </ProtectedRoute>
              } />
              <Route path="/tasks" element={
                <ProtectedRoute>
                  <TaskManager />
                </ProtectedRoute>
              } />
              <Route path="/projects" element={
                <ProtectedRoute requiredRole={['admin', 'manager']}>
                  <ProjectManagement />
                </ProtectedRoute>
              } />
              <Route path="/users" element={
                <ProtectedRoute requiredRole="admin">
                  <UserManagement />
                </ProtectedRoute>
              } />
              <Route path="/reports" element={
                <ProtectedRoute>
                  <Reports />
                </ProtectedRoute>
              } />
              <Route path="/backup" element={
                <ProtectedRoute requiredRole="admin">
                  <BackupManager />
                </ProtectedRoute>
              } />
              <Route path="/" element={<Navigate to="/dashboard" />} />
            </Routes>
          </div>
        </div>
      </Router>
    </AuthProvider>
  );
}

export default App;