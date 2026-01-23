import axios from 'axios';

// Автоматическое определение базового URL API
const getApiBaseUrl = () => {
  // Если фронтенд работает на localhost (разработка), используем localhost:8080
  if (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1') {
    return 'http://localhost:8080';
  }
  // В проде API находится на том же хосте что и фронтенд
  // В проде используем абсолютные пути в компонентах (например /api/login)
  // поэтому возвращаем пустой baseURL — запросы начнутся от корня фронтенда
  return '';
};

const api = axios.create({
  baseURL: getApiBaseUrl(),
  timeout: 30000, // Увеличиваем таймаут для VM
});

// Интерцептор для добавления токена к каждому запросу
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Интерцептор для обработки ошибок
api.interceptors.response.use(
  (response) => response,
  (error) => {
    // Проверяем, это ошибка 401 при доступе к защищённому ресурсу (не логин/регистр)
    const isAuthEndpoint = error.config?.url?.includes('/api/login') || 
                           error.config?.url?.includes('/api/register');
    
    if (error.response?.status === 401 && !isAuthEndpoint) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      window.location.href = '/login';
    }
    
    if (error.code === 'ECONNABORTED') {
      console.error('Request timeout:', error);
      throw new Error('Request timeout. Please check your connection.');
    }
    
    if (error.response) {
      // Сервер ответил с ошибкой
      console.error('API Error:', error.response.status, error.response.data);
    } else if (error.request) {
      // Запрос был сделан, но ответ не получен
      console.error('Network Error:', error.request);
      throw new Error('Network error. Please check your connection.');
    } else {
      // Что-то пошло не так при настройке запроса
      console.error('Request Error:', error.message);
    }
    
    return Promise.reject(error);
  }
);

export default api;