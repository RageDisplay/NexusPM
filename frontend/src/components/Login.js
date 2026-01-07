import React, { useState } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useNavigate } from 'react-router-dom';
import api from '../utils/api';

const Login = () => {
  const [isLogin, setIsLogin] = useState(true);
  const [formData, setFormData] = useState({
    username: '',
    password: '',
    department: ''
  });
  const [error, setError] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  const { login } = useAuth();
  const navigate = useNavigate();

  // Список доступных отделов
  const departments = [
    { value: '', label: 'Выбор отдела', disabled: true },
    { value: 'ОП', label: 'ОП' },
    { value: 'ОВ', label: 'ОВ' },
    { value: 'РП', label: 'РП' },
    { value: 'ГИП', label: 'ГИП' },
    { value: 'ПС', label: 'ПС' }
  ];

  const handleSubmit = async (e) => {
    e.preventDefault();
    
    // Валидация полей
    if (!formData.username.trim()) {
      setError('Пожалуйста, введите имя пользователя');
      return;
    }
    if (!formData.password.trim()) {
      setError('Пожалуйста, введите пароль');
      return;
    }
    if (!isLogin && !formData.department) {
      setError('Пожалуйста, выберите отдел');
      return;
    }

    setError('');
    setIsLoading(true);

    try {
      const endpoint = isLogin ? '/api/login' : '/api/register';
      const response = await api.post(endpoint, formData);
      
      if (response.data.token) {
        login(response.data.user, response.data.token);
        navigate('/dashboard');
      }
    } catch (error) {
      setError(error.response?.data?.error || 'Произошла ошибка при запросе к серверу');
    } finally {
      setIsLoading(false);
    }
  };

  const handleInputChange = (e, field) => {
    setFormData({...formData, [field]: e.target.value});
  };

  const togglePasswordVisibility = () => {
      setShowPassword(!showPassword);
    };

  return (
    <div className="login-container">
      <h2>{isLogin ? 'Авторизация' : 'Регистрация'}</h2>
      <form onSubmit={handleSubmit} className="login-form">
        <div className="form-group">
          <label>Имя пользователя</label>
          <input
            type="text"
            value={formData.username}
            onChange={(e) => handleInputChange(e, 'username')}
            disabled={isLoading}
            required
          />
        </div>
        <div className="form-group">
          <label>Пароль</label>
          <div className="password-input-container">
            <input
              type={showPassword ? "text" : "password"}
              value={formData.password}
              onChange={(e) => handleInputChange(e, 'password')}
              disabled={isLoading}
              required
            />
            <button
              type="button"
              className="password-toggle"
              onClick={togglePasswordVisibility}
              disabled={isLoading}
            >
              {showPassword ? '👁' : '👁'}
            </button>
        </div>
      </div>
        
        {/* Поле отдела только для регистрации */}
        {!isLogin && (
          <div className="form-group">
            <label>Отдел</label>
            <select
              value={formData.department}
              onChange={(e) => handleInputChange(e, 'department')}
              disabled={isLoading}
              required={!isLogin}
            >
              {departments.map(dept => (
                <option 
                  key={dept.value} 
                  value={dept.value} 
                  disabled={dept.disabled}
                >
                  {dept.label}
                </option>
              ))}
            </select>
            <small className="form-help">
              Пожалуйста, выберите свой отдел
            </small>
          </div>
        )}
        
        {error && (
          <div className="error">
            <div className="error-content">
              <span>{error}</span>
              <button 
                type="button"
                className="error-close"
                onClick={() => setError('')}
                title="Закрыть"
              >
                ✕
              </button>
            </div>
          </div>
        )}
        <button type="submit" className="btn btn-primary" disabled={isLoading}>
          {isLoading ? 'Пожалуйста, ожидайте...' : (isLogin ? 'Авторизация' : 'Регистрация')}
        </button>
      </form>
      <p>
        {isLogin ? "Нет аккаунта ? " : "Уже есть аккаунт ? "}
        <button 
          type="button" 
          className="btn-link"
          onClick={() => {
            setIsLogin(!isLogin);
            setFormData({
              username: '',
              password: '',
              department: ''
            });
            setError('');
          }}
          disabled={isLoading}
        >
          {isLogin ? 'Регистрация' : 'Авторизация'}
        </button>
      </p>
    </div>
  );
};

export default Login;