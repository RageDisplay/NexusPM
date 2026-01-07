import React, { useState, useEffect } from 'react';
import api from '../utils/api';

const ADConfigPanel = () => {
  const [config, setConfig] = useState(null);
  const [isEditing, setIsEditing] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [successMessage, setSuccessMessage] = useState('');
  const [isSyncing, setIsSyncing] = useState(false);

  const [formData, setFormData] = useState({
    enabled: false,
    directory_type: 'active_directory',
    server_url: '',
    base_dn: '',
    bind_dn: '',
    bind_password: '',
    user_search_base: '',
    user_name_attr: 'sAMAccountName',
    department_attr: 'department',
    email_attr: 'mail',
    group_search_base: '',
    sync_interval: 60
  });

  useEffect(() => {
    loadADConfig();
  }, []);

  const loadADConfig = async () => {
    try {
      setIsLoading(true);
      const response = await api.get('/api/ad-config');
      if (response.data.config) {
        setConfig(response.data.config);
        setFormData({
          enabled: response.data.config.enabled,
          directory_type: response.data.config.directory_type,
          server_url: response.data.config.server_url,
          base_dn: response.data.config.base_dn,
          bind_dn: response.data.config.bind_dn,
          bind_password: '', // Не заполняем пароль из соображений безопасности
          user_search_base: response.data.config.user_search_base,
          user_name_attr: response.data.config.user_name_attr,
          department_attr: response.data.config.department_attr,
          email_attr: response.data.config.email_attr,
          group_search_base: response.data.config.group_search_base,
          sync_interval: response.data.config.sync_interval
        });
      }
    } catch (error) {
      console.error('Error loading AD config:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleInputChange = (e) => {
    const { name, value, type, checked } = e.target;
    setFormData({
      ...formData,
      [name]: type === 'checkbox' ? checked : value
    });
  };

  const handleTestConnection = async () => {
    try {
      setError('');
      setSuccessMessage('');
      setIsLoading(true);

      await api.post('/api/ad-config/test', {
        server_url: formData.server_url,
        bind_dn: formData.bind_dn,
        bind_password: formData.bind_password
      });

      setSuccessMessage('Подключение успешно!');
    } catch (error) {
      setError(error.response?.data?.error || 'Ошибка подключения');
    } finally {
      setIsLoading(false);
    }
  };

  const handleSaveConfig = async () => {
    try {
      setError('');
      setSuccessMessage('');
      setIsLoading(true);

      await api.post('/api/ad-config', formData);
      setSuccessMessage('Конфигурация AD сохранена успешно!');
      setIsEditing(false);
      loadADConfig();
    } catch (error) {
      setError(error.response?.data?.error || 'Ошибка при сохранении конфигурации');
    } finally {
      setIsLoading(false);
    }
  };

  const handleSyncUsers = async () => {
    try {
      setError('');
      setSuccessMessage('');
      setIsSyncing(true);

      const response = await api.post('/api/ad-config/sync');
      setSuccessMessage(`Синхронизировано пользователей: ${response.data.users_synced}`);
    } catch (error) {
      setError(error.response?.data?.error || 'Ошибка при синхронизации');
    } finally {
      setIsSyncing(false);
    }
  };

  return (
    <div className="ad-config-panel">
      <h2>Конфигурация Active Directory / FreeIPA</h2>

      {error && (
        <div className="error-message">
          {error}
          <button onClick={() => setError('')} className="close-btn">✕</button>
        </div>
      )}

      {successMessage && (
        <div className="success-message">
          {successMessage}
          <button onClick={() => setSuccessMessage('')} className="close-btn">✕</button>
        </div>
      )}

      {!isEditing && config ? (
        <div className="config-display">
          <div className="config-info">
            <p><strong>Статус:</strong> {config.enabled ? '✓ Включено' : '✗ Отключено'}</p>
            <p><strong>Тип директории:</strong> {config.directory_type}</p>
            <p><strong>URL сервера:</strong> {config.server_url}</p>
            <p><strong>Base DN:</strong> {config.base_dn}</p>
            <p><strong>OU для поиска пользователей:</strong> {config.user_search_base}</p>
            <p><strong>Интервал синхронизации:</strong> {config.sync_interval} минут</p>
          </div>

          <div className="config-buttons">
            <button
              onClick={() => setIsEditing(true)}
              className="btn btn-secondary"
              disabled={isLoading}
            >
              Редактировать
            </button>
            <button
              onClick={handleSyncUsers}
              className="btn btn-primary"
              disabled={isLoading || isSyncing || !config.enabled}
            >
              {isSyncing ? 'Синхронизация...' : 'Синхронизировать пользователей'}
            </button>
          </div>
        </div>
      ) : (
        <form className="ad-config-form">
          <div className="form-section">
            <h3>Основные параметры</h3>

            <div className="form-group">
              <label>
                <input
                  type="checkbox"
                  name="enabled"
                  checked={formData.enabled}
                  onChange={handleInputChange}
                  disabled={isLoading}
                />
                <span>Включить интеграцию с AD/FreeIPA</span>
              </label>
            </div>

            <div className="form-group">
              <label>Тип директории:</label>
              <select
                name="directory_type"
                value={formData.directory_type}
                onChange={handleInputChange}
                disabled={isLoading}
              >
                <option value="active_directory">Active Directory (AD)</option>
                <option value="freeipa">FreeIPA</option>
              </select>
            </div>

            <div className="form-group">
              <label>URL LDAP сервера:</label>
              <input
                type="text"
                name="server_url"
                placeholder="ldap://192.168.1.100:389"
                value={formData.server_url}
                onChange={handleInputChange}
                disabled={isLoading}
              />
              <small>Пример: ldap://dc.example.com:389 или ldaps://dc.example.com:636</small>
            </div>

            <div className="form-group">
              <label>Base DN:</label>
              <input
                type="text"
                name="base_dn"
                placeholder="dc=example,dc=com"
                value={formData.base_dn}
                onChange={handleInputChange}
                disabled={isLoading}
              />
            </div>
          </div>

          <div className="form-section">
            <h3>Учётные данные сервиса</h3>

            <div className="form-group">
              <label>Bind DN (DN сервисного аккаунта):</label>
              <input
                type="text"
                name="bind_dn"
                placeholder="cn=service_account,cn=users,dc=example,dc=com"
                value={formData.bind_dn}
                onChange={handleInputChange}
                disabled={isLoading}
              />
            </div>

            <div className="form-group">
              <label>Пароль сервисного аккаунта:</label>
              <input
                type="password"
                name="bind_password"
                placeholder="Пароль"
                value={formData.bind_password}
                onChange={handleInputChange}
                disabled={isLoading}
              />
            </div>

            <button
              type="button"
              onClick={handleTestConnection}
              className="btn btn-secondary"
              disabled={isLoading || !formData.server_url || !formData.bind_dn}
            >
              Проверить подключение
            </button>
          </div>

          <div className="form-section">
            <h3>Параметры поиска пользователей</h3>

            <div className="form-group">
              <label>OU для поиска пользователей:</label>
              <input
                type="text"
                name="user_search_base"
                placeholder="ou=Users,dc=example,dc=com"
                value={formData.user_search_base}
                onChange={handleInputChange}
                disabled={isLoading}
              />
            </div>

            <div className="form-group">
              <label>Атрибут имени пользователя:</label>
              <input
                type="text"
                name="user_name_attr"
                placeholder="sAMAccountName (для AD) или uid (для FreeIPA)"
                value={formData.user_name_attr}
                onChange={handleInputChange}
                disabled={isLoading}
              />
              <small>AD: sAMAccountName, FreeIPA: uid</small>
            </div>

            <div className="form-group">
              <label>Атрибут отдела:</label>
              <input
                type="text"
                name="department_attr"
                placeholder="department"
                value={formData.department_attr}
                onChange={handleInputChange}
                disabled={isLoading}
              />
            </div>

            <div className="form-group">
              <label>Атрибут email:</label>
              <input
                type="text"
                name="email_attr"
                placeholder="mail"
                value={formData.email_attr}
                onChange={handleInputChange}
                disabled={isLoading}
              />
            </div>

            <div className="form-group">
              <label>OU для поиска групп (опционально):</label>
              <input
                type="text"
                name="group_search_base"
                placeholder="ou=Groups,dc=example,dc=com"
                value={formData.group_search_base}
                onChange={handleInputChange}
                disabled={isLoading}
              />
            </div>

            <div className="form-group">
              <label>Интервал синхронизации (минуты):</label>
              <input
                type="number"
                name="sync_interval"
                value={formData.sync_interval}
                onChange={handleInputChange}
                disabled={isLoading}
                min="1"
              />
            </div>
          </div>

          <div className="form-actions">
            <button
              type="button"
              onClick={handleSaveConfig}
              className="btn btn-primary"
              disabled={isLoading}
            >
              {isLoading ? 'Сохранение...' : 'Сохранить конфигурацию'}
            </button>
            {config && (
              <button
                type="button"
                onClick={() => {
                  setIsEditing(false);
                  loadADConfig();
                }}
                className="btn btn-secondary"
                disabled={isLoading}
              >
                Отмена
              </button>
            )}
          </div>
        </form>
      )}
    </div>
  );
};

export default ADConfigPanel;
