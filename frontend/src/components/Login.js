import React, { useState } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useNavigate } from 'react-router-dom';
import api from '../utils/api';

const Login = () => {
  const [isLogin, setIsLogin] = useState(true);
  const [authType, setAuthType] = useState('auto'); // 'auto', 'local', 'ad'
  const [adEnabled, setAdEnabled] = useState(false);
  const [showDepartmentModal, setShowDepartmentModal] = useState(false);
  const [showPasswordResetModal, setShowPasswordResetModal] = useState(false);
  const [resetPasswordData, setResetPasswordData] = useState({
    token: null,
    user: null,
    newPassword: '',
    confirmPassword: ''
  });
  const [departmentModalData, setDepartmentModalData] = useState({
    token: null,
    user: null,
    selectedDepartment: ''
  });
  const [formData, setFormData] = useState({
    username: '',
    password: '',
    department: ''
  });
  const [error, setError] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [showForgotForm, setShowForgotForm] = useState(false);
  const [forgotRequestMessage, setForgotRequestMessage] = useState('');
  const [forgotRequestSubmitting, setForgotRequestSubmitting] = useState(false);
  const [forgotRequestSent, setForgotRequestSent] = useState(false);

  const { login } = useAuth();
  const navigate = useNavigate();

  const [departments, setDepartments] = React.useState([
    { value: '', label: 'Выбор отдела', disabled: true }
  ]);

  // Проверяем доступность AD при загрузке и загружаем отделы
  React.useEffect(() => {
    checkADAvailable();
    fetchDepartments();
  }, []);

  const fetchDepartments = async () => {
    try {
      const response = await api.get('/api/auth/departments');
      if (response.data.departments) {
        setDepartments(response.data.departments);
      }
    } catch (error) {
      console.error('Error fetching departments:', error);
      // Используем дефолтный список если ошибка
      setDepartments([
        { value: '', label: 'Выбор отдела', disabled: true },
        { value: 'ОП', label: 'ОП' },
        { value: 'ОВ', label: 'ОВ' },
        { value: 'РП', label: 'РП' },
        { value: 'ГИП', label: 'ГИП' },
        { value: 'ПС', label: 'ПС' }
      ]);
    }
  };

  const checkADAvailable = async () => {
    try {
      const response = await api.get('/api/ad-config/status');
      if (response.data.enabled) {
        setAdEnabled(true);
      }
    } catch (error) {
      // AD не настроена
      setAdEnabled(false);
    }
  };

  const handleDepartmentModalSubmit = async () => {
    if (!departmentModalData.selectedDepartment) {
      setError('Пожалуйста, выберите отдел');
      return;
    }

    setError('');
    setIsLoading(true);

    try {
      console.log('Submitting department:', departmentModalData.selectedDepartment);
      console.log('Using token:', departmentModalData.token);
      
      // Обновляем отдел на сервере используя публичный эндпоинт (не требует авторизации)
      const response = await api.post('/api/auth/set-department', {
        token: departmentModalData.token,
        department: departmentModalData.selectedDepartment
      });
      
      console.log('Department update response:', response.data);

      // Используем обновленные данные и новый токен из ответа сервера
      const updatedUser = response.data.user || {
        ...departmentModalData.user,
        department: departmentModalData.selectedDepartment
      };
      
      const newToken = response.data.token || departmentModalData.token;

      console.log('Logging in with updated user:', updatedUser);
      login(updatedUser, newToken);
      setShowDepartmentModal(false);
      navigate('/dashboard');
    } catch (error) {
      console.error('Error updating department:', error);
      setError(error.response?.data?.error || 'Ошибка при обновлении отдела');
    } finally {
      setIsLoading(false);
    }
  };

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
      const payload = { ...formData };
      
      // Если это логин, добавляем тип аутентификации
      if (isLogin) {
        payload.auth_type = authType;
      }

      const response = await api.post(endpoint, payload);
      
      if (response.data.token) {
        // Проверяем, требуется ли сброс пароля
        const passwordResetRequired = isLogin && response.data.user.password_reset_required;
        
        // Если это пользователь из AD без отдела, показываем модальное окно
        const isADUserWithoutDepartment = isLogin && 
          response.data.user.is_ad_user && 
          (!response.data.user.department || response.data.user.department === '');
        
        console.log('Login response user:', response.data.user);
        console.log('Is AD user without department:', isADUserWithoutDepartment);
        console.log('Password reset required:', passwordResetRequired);
        
        if (passwordResetRequired) {
          setResetPasswordData({
            token: response.data.token,
            user: response.data.user,
            newPassword: '',
            confirmPassword: ''
          });
          setShowPasswordResetModal(true);
        } else if (isADUserWithoutDepartment) {
          setDepartmentModalData({
            token: response.data.token,
            user: response.data.user,
            selectedDepartment: ''
          });
          setShowDepartmentModal(true);
        } else {
          login(response.data.user, response.data.token);
          navigate('/dashboard');
        }
      }
    } catch (error) {
      setError(error.response?.data?.error || 'Произошла ошибка при запросе к серверу');
    } finally {
      setIsLoading(false);
    }
  };

  const submitForgotRequest = async () => {
    if (!formData.username) {
      setError('Введите имя пользователя перед отправкой запроса');
      return;
    }
    setForgotRequestSubmitting(true);
    try {
      await api.post('/api/password-reset-requests', {
        username: formData.username,
        message: forgotRequestMessage
      });
      setForgotRequestSent(true);
      setShowForgotForm(false);
    } catch (err) {
      setError(err.response?.data?.error || 'Ошибка отправки заявки');
    } finally {
      setForgotRequestSubmitting(false);
    }
  };

  const handleInputChange = (e, field) => {
    setFormData({...formData, [field]: e.target.value});
  };

  const togglePasswordVisibility = () => {
      setShowPassword(!showPassword);
    };

  const handlePasswordResetModalSubmit = async () => {
    setError('');

    if (!resetPasswordData.newPassword) {
      setError('Пожалуйста, введите новый пароль');
      return;
    }

    if (resetPasswordData.newPassword.length < 6) {
      setError('Пароль должен содержать минимум 6 символов');
      return;
    }

    if (resetPasswordData.newPassword !== resetPasswordData.confirmPassword) {
      setError('Пароли не совпадают');
      return;
    }

    setIsLoading(true);

    try {
      // Используем временный токен для установки нового пароля
      const config = {
        headers: {
          'Authorization': `Bearer ${resetPasswordData.token}`
        }
      };

      await api.post('/api/auth/set-new-password', {
        new_password: resetPasswordData.newPassword
      }, config);

      // После успешного сброса входим в систему
      login(resetPasswordData.user, resetPasswordData.token);
      setShowPasswordResetModal(false);
      navigate('/dashboard');
    } catch (error) {
      console.error('Error resetting password:', error);
      setError(error.response?.data?.error || 'Ошибка при установке нового пароля');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="login-container">
      <h2>{isLogin ? 'Авторизация' : 'Регистрация'}</h2>
      
      {/* Выбор типа аутентификации для логина */}
      {isLogin && adEnabled && (
        <div className="auth-type-selector">
          <div className="auth-type-options">
            <label className="radio-label">
              <input
                type="radio"
                name="authType"
                value="auto"
                checked={authType === 'auto'}
                onChange={(e) => setAuthType(e.target.value)}
                disabled={isLoading}
              />
              <span>Автоматический выбор</span>
            </label>
            <label className="radio-label">
              <input
                type="radio"
                name="authType"
                value="ad"
                checked={authType === 'ad'}
                onChange={(e) => setAuthType(e.target.value)}
                disabled={isLoading}
              />
              <span>Вход через домен AD/FreeIPA</span>
            </label>
            <label className="radio-label">
              <input
                type="radio"
                name="authType"
                value="local"
                checked={authType === 'local'}
                onChange={(e) => setAuthType(e.target.value)}
                disabled={isLoading}
              />
              <span>Локальный вход</span>
            </label>
          </div>
        </div>
      )}

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
        {forgotRequestSent && (
          <div className="success-message" style={{marginTop: '12px'}}>
            <div className="error-content">
              <span>Заявка отправлена. Администратор свяжется с вами.</span>
            </div>
          </div>
        )}
        {error && isLogin && (authType === 'local' || authType === 'auto') && !forgotRequestSent && (
          <div style={{marginTop: '8px'}}>
            {!showForgotForm ? (
              <button type="button" className="btn-link" onClick={() => setShowForgotForm(true)}>Забыли пароль? Отправить запрос администратору</button>
            ) : (
              <div style={{marginTop: '8px'}}>
                <textarea
                  placeholder="Дополнительная информация (необязательно)"
                  value={forgotRequestMessage}
                  onChange={(e) => setForgotRequestMessage(e.target.value)}
                  style={{width: '100%', minHeight: '80px'}}
                />
                <div style={{marginTop: '8px'}}>
                  <button className="btn btn-primary" type="button" onClick={submitForgotRequest} disabled={forgotRequestSubmitting}>{forgotRequestSubmitting ? 'Отправка...' : 'Отправить заявку'}</button>
                  <button className="btn btn-cancel" type="button" style={{marginLeft: '8px'}} onClick={() => setShowForgotForm(false)}>Отмена</button>
                </div>
              </div>
            )}
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

      {/* Модальное окно для установки нового пароля после сброса */}
      {showPasswordResetModal && (
        <div className="modal-overlay">
          <div className="modal-dialog">
            <div className="modal-header">
              <h3>Установка нового пароля</h3>
            </div>
            <div className="modal-body">
              <p>Администратор потребовал смену вашего пароля. Пожалуйста, установите новый пароль:</p>
              <div className="form-group">
                <label htmlFor="new-password">Новый пароль</label>
                <input
                  type="password"
                  id="new-password"
                  value={resetPasswordData.newPassword}
                  onChange={(e) => setResetPasswordData({
                    ...resetPasswordData,
                    newPassword: e.target.value
                  })}
                  placeholder="Введите новый пароль"
                  disabled={isLoading}
                  minLength="6"
                />
              </div>
              <div className="form-group">
                <label htmlFor="confirm-password">Подтвердите пароль</label>
                <input
                  type="password"
                  id="confirm-password"
                  value={resetPasswordData.confirmPassword}
                  onChange={(e) => setResetPasswordData({
                    ...resetPasswordData,
                    confirmPassword: e.target.value
                  })}
                  placeholder="Подтвердите новый пароль"
                  disabled={isLoading}
                  minLength="6"
                />
              </div>
              {error && (
                <div className="error">
                  <div className="error-content">
                    <span>{error}</span>
                    <button 
                      type="button"
                      className="error-close"
                      onClick={() => setError('')}
                    >
                      ✕
                    </button>
                  </div>
                </div>
              )}
            </div>
            <div className="modal-footer">
              <button
                type="button"
                className="btn btn-primary"
                onClick={handlePasswordResetModalSubmit}
                disabled={isLoading}
              >
                {isLoading ? 'Установка пароля...' : 'Установить пароль'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Модальное окно для выбора отдела при первом входе пользователя из AD */}
      {showDepartmentModal && (
        <div className="modal-overlay">
          <div className="modal-dialog">
            <div className="modal-header">
              <h3>Выберите ваш отдел</h3>
            </div>
            <div className="modal-body">
              <p>Ваш аккаунт был импортирован из системы домена. Пожалуйста, выберите свой отдел:</p>
              <div className="form-group">
                <select
                  value={departmentModalData.selectedDepartment}
                  onChange={(e) => setDepartmentModalData({
                    ...departmentModalData,
                    selectedDepartment: e.target.value
                  })}
                  disabled={isLoading}
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
              </div>
              {error && (
                <div className="error">
                  <div className="error-content">
                    <span>{error}</span>
                    <button 
                      type="button"
                      className="error-close"
                      onClick={() => setError('')}
                    >
                      ✕
                    </button>
                  </div>
                </div>
              )}
            </div>
            <div className="modal-footer">
              <button
                type="button"
                className="btn btn-primary"
                onClick={handleDepartmentModalSubmit}
                disabled={isLoading}
              >
                {isLoading ? 'Сохранение...' : 'Продолжить'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default Login;