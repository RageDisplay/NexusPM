# NexusPM - Система управления проектами и задачами

Комплексная система управления проектами, задачами и ресурсами с поддержкой нескольких уровней доступа, интеграцией с Active Directory/FreeIPA и автоматическими отчетами.

## Быстрые ссылки

**Версия:** 2.1.0 ✨ (Multi-Department Manager Access)  
**Стек:** Go + React + SQLite + Docker  

**Содержание:**
- [Быстрый старт](#быстрый-старт)
- [Архитектура](#архитектура-приложения)
- [Для пользователей](#для-обычных-пользователей)
- [Для администраторов](#для-администраторов-системы)
- [Для разработчиков](#для-разработчиков)

---

## Быстрый старт

### Установка с Docker (за 3 команды)

```bash
# 1. Клонируйте/откройте проект
cd NexusPM

# 2. Запустите контейнеры
docker-compose up -d

# 3. Откройте браузер
# http://localhost:7000
# Логин: admin | Пароль: main12!@
```

## Архитектура приложения

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          WEB BROWSER (HTTP)                              │
└──────────────────────────┬──────────────────────────────────────────────┘
                           │
         ┌─────────────────▼──────────────────┐
         │      FRONTEND (React.js)            │
         │  ┌───────────────────────────────┐  │
         │  │  Components:                   │  │
         │  │  • Dashboard                   │  │
         │  │  • TaskManager                 │  │
         │  │  • ProjectManagement           │  │
         │  │  • UserManagement              │  │
         │  │  • Reports (Excel Export)      │  │
         │  │  • ADConfigPanel               │  │
         │  │  • Notifications               │  │
         │  └───────────────────────────────┘  │
         │  State: AuthContext + localStorage  │
         │  Base URL: http://localhost:7000    │
         └──────────────┬──────────────────────┘
                        │ REST API (JSON)
         ┌──────────────▼──────────────────┐
         │    BACKEND (Go + Gin)            │
         │  ┌───────────────────────────────┐  │
         │  │  API Handlers:                  │  │
         │  │  • Auth (JWT)                   │  │
         │  │  • Tasks CRUD                   │  │
         │  │  • Projects & Assignment        │  │
         │  │  • Users Management             │  │
         │  │  • Reports (Excel)              │  │
         │  │  • Dashboard & Statistics       │  │
         │  │  • AD/LDAP Sync                 │  │
         │  │  • Backup/Restore               │  │
         │  └───────────────────────────────┘  │
         │  ┌───────────────────────────────┐  │
         │  │  Middleware:                    │  │
         │  │  • JWT Auth                     │  │
         │  │  • Role-based Access Control    │  │
         │  │  • CORS Headers                 │  │
         │  └───────────────────────────────┘  │
         │  ┌───────────────────────────────┐  │
         │  │  Integrations:                  │  │
         │  │  • LDAP/Active Directory        │  │
         │  │  • FreeIPA                      │  │
         │  │  • Excel Export (excelize)      │  │
         │  └───────────────────────────────┘  │
         │  Base URL: http://localhost:8080    │
         └──────────────┬──────────────────────┘
                        │ SQL Queries
         ┌──────────────▼──────────────────┐
         │   DATABASE (SQLite3)             │
         │  ┌───────────────────────────────┐  │
         │  │  Tables:                       │  │
         │  │  • users (auth, roles, dept)   │  │
         │  │  • tasks (task management)     │  │
         │  │  • projects (projects)         │  │
         │  │  • user_projects (assignment)  │  │
         │  │  • notifications               │  │
         │  │  • password_reset_requests     │  │
         │  │  • ad_config (AD settings)     │  │
         │  │  • activity_log                │  │
         │  └───────────────────────────────┘  │
         │  File: /backend/data/tasks.db       │
         └──────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────┐
│                    DOCKER COMPOSE ORCHESTRATION                           │
│  • nginx proxy (7000 → frontend)                                          │
│  • Frontend container: Node.js + React                                    │
│  • Backend container: Go runtime + SQLite                                 │
│  • Volume: ./backend/data/tasks.db (persistent)                           │
└──────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────┐
│                    ВНЕШНИЕ СЕРВИСЫ (ОПЦИОНАЛЬНО)                          │
│  ┌─────────────────────┐         ┌──────────────────┐                    │
│  │ LDAP/AD Server      │         │ SMTP (Email)     │                    │
│  │ (User Sync)         │         │ (Notifications)  │                    │
│  └─────────────────────┘         └──────────────────┘                    │
└──────────────────────────────────────────────────────────────────────────┘
```

### Поток данных

```
1. АУТЕНТИФИКАЦИЯ
   Browser → Login Form → Backend (/api/login) → JWT Token + User Data → localStorage

2. ЗАЩИТА ЗАПРОСОВ
   Browser → API Call → Authorization Header (Bearer Token) → Backend (Verify)

3. УПРАВЛЕНИЕ ЗАДАЧАМИ
   User → TaskManager UI → POST /api/tasks → Backend (Validate + DB) → SQL Insert

4. СИНХРОНИЗАЦИЯ AD
   Admin → ADConfigPanel → POST /api/ad-config/sync → LDAP Server → Create/Update Users

5. ЭКСПОРТ ОТЧЕТОВ
   User → Reports UI → GET /api/reports/* → Backend (Excel Generate) → Download XLSX
```

---

# ДЛЯ ОБЫЧНЫХ ПОЛЬЗОВАТЕЛЕЙ

## Обзор системы

NexusPM помогает организовать работу в проектах, отслеживать задачи, планировать нагрузку и синхронизировать данные через отчеты.

### Основные возможности:

**Управление задачами** - создание, редактирование, отслеживание прогресса  
**Планирование нагрузки** - контроль часов работы и загруженности в месяц  
**Проекты** - назначение на проекты своего отдела  
**Отчеты** - экспорт задач в Excel с детальной информацией  
**Уведомления** - управление заявками на сброс пароля

## Первый вход в систему

### Регистрация (если требуется)

1. Откройте страницу входа
2. Нажмите "Создать аккаунт" или ссылку регистрации
3. Введите:
   - **Логин** - уникальный идентификатор (только латиница и цифры)
   - **Пароль** - минимум 6 символов
   - **Отдел** - выберите свой отдел из списка
4. Нажмите "Зарегистрироваться"

### Вход в систему

1. Введите логин и пароль
2. Нажмите "Войти"
3. Если включена интеграция с Active Directory - данные синхронизируются автоматически

### Первый вход после сброса пароля

После сброса пароля администратором система требует установить новый пароль:
1. Введите временный пароль (полученный от администратора)
2. Введите новый пароль (минимум 6 символов)
3. Подтвердите новый пароль
4. Нажмите "Установить пароль"

## Работа с задачами

### Просмотр задач

На вкладке **"Задачи"** вы видите список всех ваших задач с информацией:

| Колонка | Описание |
|---------|---------|
| **Название** | Название проекта/задачи |
| **Прогресс** | % выполнения (0-100%) |
| **За неделю** | Что было сделано на этой неделе |
| **Планирование** | Что планируется делать дальше |
| **Помощь** | Где нужна помощь/что блокирует |
| **Часов/неделю** | Скольок часов в неделю требует задача |
| **Нагрузка/месяц** | Загруженность в процентах в месяц |

### Создание новой задачи (для менеджеров и выше)

1. В разделе "Задачи" найдите форму **"Добавить новую задачу"**
2. Заполните поля:
   - **Проект** - выберите из списка проектов вашего отдела
   - **Сотрудник** - кому назначить задачу
   - **Описание задачи** - подробное описание
   - **За неделю** - что нужно сделать на текущей неделе
   - **Планирование** - план на следующий период
   - **Помощь** - какая помощь требуется
   - **Часов/неделю** - количество часов
   - **Нагрузка** - загруженность (%)
3. Нажмите **"Добавить"**

### Редактирование задачи

1. Найдите задачу в списке
2. Нажмите на неё для раскрытия деталей
3. Нажмите кнопку **"Редактировать"** или просто отредактируйте поля
4. Обновите информацию
5. Сохраните изменения

### Удаление задачи

1. Найдите задачу
2. Нажмите кнопку **"Удалить"** или значок 🗑️
3. Подтвердите удаление в диалоговом окне

## Работа с проектами (для менеджеров и выше)

### Просмотр проектов

На вкладке **"Проекты"** видны:
- Все проекты вашего отдела (для менеджеров)
- Все проекты системы (для админов)

### Создание проекта

1. В разделе "Проекты" найдите форму **"Создать новый проект"**
2. Заполните:
   - **Название** - уникальное имя проекта
   - **Описание** - детали, требования, результаты
3. Нажмите **"✓ Создать проект"**
4. Отдел устанавливается автоматически (ваш отдел для менеджеров)

### Назначение сотрудников на проект

1. Выберите проект из списка
2. В разделе **"Добавить сотрудника"** нажмите выпадающий список
3. Выберите сотрудника (менеджеры могут выбирать только из своего отдела)
4. Нажмите **"Добавить"**

**Примечание:** Менеджер может добавлять:
- Простых сотрудников своего отдела
- Других менеджеров из своего отдела
- Админов того же отдела

### Удаление сотрудника из проекта

1. В списке "Назначенные сотрудники" найдите человека
2. Нажмите кнопку **"✕ Удалить"** рядом с именем
3. Подтвердите удаление

### Удаление проекта

1. Выберите проект
2. Нажмите кнопку **"🗑️ Удалить проект"**
3. Подтвердите удаление (это необратимо)

## Экспорт отчетов в Excel

### Экспортирование ваших задач

1. Перейдите на вкладку **"Отчеты"**
2. Нажмите **"Экспортировать мои задачи"**
3. Начнется скачивание файла `my_tasks.xlsx`
4. Откройте файл в Excel - содержит все ваши задачи с деталями

### Экспортирование задач отдела (для менеджеров)

1. На вкладке "Отчеты" нажмите **"Экспортировать задачи отдела"**
2. Скачается файл `department_stats.xlsx`
3. Содержит полную статистику по всем сотрудникам вашего отдела

### Экспортирование всех задач (для админов)

1. На вкладке "Отчеты" нажмите **"Экспортировать все задачи"**
2. Скачается файл `all_projects.xlsx`
3. Содержит полную информацию по всем проектам и задачам системы

## Управление профилем

### Просмотр профиля

1. На вкладке **"Профиль"** (или в меню) видно:
   - Ваше имя пользователя
   - Текущая роль (Сотрудник/Менеджер/Администратор)
   - Ваш отдел

### Смена пароля

1. На вкладке "Профиль" найдите **"Сменить пароль"**
2. Введите текущий пароль
3. Введите новый пароль (минимум 6 символов)
4. Подтвердите новый пароль
5. Нажмите **"Сменить"**

---

# ДЛЯ АДМИНИСТРАТОРОВ СИСТЕМЫ

## Обзор администрирования

Администратор системы имеет полный доступ ко всем функциям:
- Управление пользователями (создание, удаление, изменение ролей)
- Управление всеми проектами и задачами
- Конфигурация интеграции с Active Directory / FreeIPA
- Создание резервных копий базы данных
- Просмотр всех данных системы

## Установка и запуск системы

### Требования

**Системные требования:**
- **ОС:** Windows, Linux, macOS (все платформы)
- **Docker** (рекомендуется):
  - Docker 20.10+
  - Docker Compose 1.29+
  - 500 MB свободного места
  
**Либо локальная установка:**
- **Go:** 1.20 или выше
- **Node.js:** 16.0 или выше (LTS)
- **npm:** 8.0 или выше
- **SQLite3:** встроена в Go

**Порты:**
- **Frontend:** 7000 (или 3000 при локальном запуске, 80 при сборке из бандла)
- **Backend:** 8080
- **LDAP:** 389 (если используется AD/FreeIPA)

### Переменные окружения

#### Backend (.env файл)

```bash
# Обязательные
ENCRYPTION_KEY=<32-символная строка в hex формате>
# Сгенерировать: python3 scripts/generate_encryption_key.py

# Опциональные (есть значения по умолчанию)
GIN_MODE=release                # release или debug
DB_PATH=/app/data/tasks.db      # Путь к БД (в Docker)
JWT_EXPIRY=24h                  # Время жизни JWT токена
```

#### Frontend (.env файл - опционально)

```bash
REACT_APP_API_URL=http://localhost:8080
REACT_APP_ENV=production
```

### Установка с Docker (рекомендуется)

**Шаг 1: Подготовка переменных окружения**

```bash
# Перейдите в backend
cd backend

# Сгенерируйте ключ шифрования
python3 scripts/generate_encryption_key.py
# или (Windows)
python scripts/generate_encryption_key.py

# Будет создан файл backend/.env с ENCRYPTION_KEY
```

**Шаг 2: Запуск контейнеров**

```bash
# Вернитесь в корень проекта
cd ..

# Запустите Docker Compose
docker-compose up -d --build

# Проверьте статус контейнеров
docker-compose ps
```

**Шаг 3: Первый вход**

```
URL: http://localhost:7000
Логин: admin
Пароль: main12!@
```

**Команды управления:**

```bash
# Остановить контейнеры
docker-compose stop

# Перезагрузить контейнеры
docker-compose restart

# Просмотреть логи
docker-compose logs -f backend    # Логи backend
docker-compose logs -f frontend   # Логи frontend

# Удалить контейнеры (данные сохранятся)
docker-compose down

# Полная очистка (включая базу данных)
docker-compose down -v
```

### Запуск без Docker (локальная разработка)

**Backend (Go)**

```bash
# Перейдите в backend
cd backend

# Сгенерируйте ENCRYPTION_KEY (если еще не сделали)
python3 scripts/generate_encryption_key.py

# Установите зависимости
go mod download

# Запустите сервер
go run main.go

# Сервер стартанет на http://localhost:8080
```

**Frontend (React)**

В **новом терминале**:

```bash
# Перейдите в frontend
cd frontend

# Установите зависимости
npm install

# Запустите разработчика сервер
npm start

# Откроется на http://localhost:3000
```

**Проверка готовности**

```bash
# Backend готов, когда видите:
# Server starting on :8080

# Frontend готов, когда браузер откроет:
# http://localhost:3000

# Проверка API подключения:
curl http://localhost:8080/api/ad-config/status
```

---

# ДЛЯ АДМИНИСТРАТОРОВ СИСТЕМЫ

## Обзор администрирования

Администратор системы имеет полный доступ ко всем функциям:
- Управление пользователями (создание, удаление, изменение ролей)
- Управление всеми проектами и задачами
- Конфигурация интеграции с Active Directory / FreeIPA
- Создание резервных копий базы данных
- Просмотр всех данных системы

## Безопасность

### Аутентификация и авторизация

**JWT Token (JSON Web Token):**
- Выдается при входе на 24 часа
- Хранится в `localStorage` браузера
- Отправляется в заголовке `Authorization: Bearer {token}`
- Содержит: user_id, username, role, department, exp time
- Проверяется на каждый защищенный запрос

**Пароли:**
- Хешируются через bcrypt (не хранятся в открытом виде)
- Минимальная длина: 6 символов (рекомендуется 12+)
- При сбросе генерируется криптографически стойкий временный пароль
- Пользователь обязан установить новый пароль после сброса

**Роли и доступ:**
```
USER (обычный сотрудник)
├─ Видит только свои задачи
├─ Может управлять только своими задачами
└─ Может экспортировать только свои задачи

MANAGER (менеджер)
├─ Видит все задачи своего отдела
├─ Может управлять задачами своего отдела
├─ Может создавать/назначать проекты для отдела
├─ Может управлять только пользователями своего отдела
└─ Видит статистику своего отдела

ADMIN (администратор)
├─ Полный доступ ко всему
├─ Управление всеми пользователями
├─ Конфигурация системы
└─ Интеграция с внешними системами
```

### Шифрование

**ENCRYPTION_KEY:**
- Используется для шифрования чувствительных данных
- 32 символа в hexadecimal формате
- **ОБЯЗАТЕЛЬНО** установить перед первым запуском
- **ВАЖНО:** Не делитесь ключом, храните в безопасности
- Сгенерировать: `python3 backend/scripts/generate_encryption_key.py`

### HTTPS (для продакшена)

**Текущая конфигурация (разработка):**
- HTTP (не защищен)
- CORS включен для всех источников (для разработки)

**Для прода необходимо:**
1. Добавить SSL сертификаты
2. Сконфигурировать nginx (см. frontend/nginx.conf)
3. Ограничить CORS по источникам
4. Включить HTTPS только

## Управление пользователями

### Просмотр всех пользователей

1. Откройте вкладку **"Пользователи"**
2. Видна таблица со всеми пользователями системы

### Создание пользователя

**Вариант 1: Самостоятельная регистрация**
- Пользователь сам заходит и регистрируется
- После регистрации имеет роль `user`

**Вариант 2: Создание администратором (через интеграцию AD)**
- При включенной интеграции с Active Directory пользователи синхронизируются автоматически

### Изменение роли пользователя

1. На вкладке "Пользователи" найдите пользователя
2. В колонке "Role" выберите новую роль:
   - **user** - простой сотрудник
   - **manager** - менеджер
   - **admin** - администратор
3. Изменение применяется сразу

### Сброс пароля пользователя

1. Найдите пользователя
2. Нажмите **"Reset Пароль"** (желтая кнопка)
3. Подтвердите действие
4. Система сгенерирует временный пароль
5. Скопируйте пароль и передайте пользователю

### Удаление пользователя

1. Найдите пользователя в таблице
2. Нажмите красную кнопку **"Delete"**
3. Подтвердите удаление
4. **Условие:** Пользователь не должен иметь открытых задач

## Интеграция с Active Directory / FreeIPA

### Включение интеграции AD

1. Перейдите на вкладку **"Конфигурация AD"**
2. Установите флаг **"Включить интеграцию с AD/FreeIPA"**

### Настройка параметров подключения

1. **Тип директории:** Active Directory (AD) или FreeIPA
2. **URL LDAP сервера:** `ldap://192.168.1.100:389`
3. **Base DN:** `dc=example,dc=com`
4. **Bind DN:** `cn=service_account,cn=users,dc=example,dc=com`
5. **Пароль сервисного аккаунта:** введите пароль
6. **OU для поиска пользователей:** `ou=Users,dc=example,dc=com`
7. **Атрибут имени:** `sAMAccountName` (AD) или `uid` (FreeIPA)
8. **Интервал синхронизации:** количество минут (пока заглушка)

### Проверка подключения

1. После заполнения параметров нажмите **"Проверить подключение"**
2. Если успешно - увидите сообщение "Подключение успешно!"
3. Нажмите **"Сохранить конфигурацию"**

### Синхронизация пользователей

1. Нажмите **"Синхронизировать пользователей"**
2. Система загрузит всех пользователей из AD/FreeIPA
3. Синхронизация происходит автоматически по установленному интервалу

## Резервное копирование и восстановление

### Создание резервной копии

1. На вкладке "Пользователи" нажмите **"Скачать резервную копию"**
2. Начнется скачивание файла `backup.db`
3. Сохраните файл в безопасном месте

### Восстановление из резервной копии

1. На вкладке "Пользователи" нажмите **"Выберите файл"**
2. Выберите ранее сохраненный `backup.db`
3. Нажмите **"Восстановить"**
4. **Внимание:** Это заменит текущую БД!

---

# ДЛЯ РАЗРАБОТЧИКОВ

## Архитектура системы

```
CLIENT (React)
    ↓ HTTP API (REST)
BACKEND (Go/Gin)
    ↓ SQL Queries
DATABASE (SQLite)
```

## Структура проекта

```
NexusPM/
├── backend/                    # Go приложение
│   ├── main.go                 # Точка входа
│   ├── go.mod                  # Зависимости
│   ├── handlers/               # API обработчики
│   │   ├── auth.go
│   │   ├── tasks.go
│   │   ├── projects.go
│   │   ├── users.go
│   │   ├── reports.go
│   │   ├── password_requests.go
│   │   ├── ad_config.go
│   │   └── backup.go
│   ├── database/               # Работа с БД
│   │   ├── database.go
│   │   ├── auth.go
│   │   └── check.go
│   ├── middleware/             # Middleware
│   │   └── auth.go
│   ├── models/                 # Структуры данных
│   │   └── models.go
│   ├── ldap/                   # Интеграция LDAP
│   │   └── ldap.go
│   └── Dockerfile
│
├── frontend/                   # React приложение
│   ├── src/
│   │   ├── index.js
│   │   ├── App.js
│   │   ├── App.css
│   │   ├── components/
│   │   │   ├── TaskManager.js
│   │   │   ├── ProjectManagement.js
│   │   │   ├── UserManagement.js
│   │   │   ├── Reports.js
│   │   │   ├── Login.js
│   │   │   ├── Dashboard.js
│   │   │   └── ADConfigPanel.js
│   │   ├── contexts/
│   │   │   └── AuthContext.js
│   │   └── utils/
│   │       └── api.js
│   ├── package.json
│   └── Dockerfile
│
├── docker-compose.yml
├── README.md
└── data/                       # Данные (при запуске)
    └── tasks.db                # SQLite база

```

## Модель данных

### Таблица: users

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash TEXT,
    role VARCHAR(20) NOT NULL DEFAULT 'user',
    department VARCHAR(100),
    is_ad_user BOOLEAN DEFAULT 0,
    ad_user_id VARCHAR(255),
    password_reset_required BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Таблица: tasks

```sql
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    progress INTEGER DEFAULT 0,
    hours_per_week DECIMAL(10,2) DEFAULT 0,
    load_per_month INTEGER DEFAULT 0,
    weekly_info TEXT,
    planning TEXT,
    help_needed TEXT,
    user_id INTEGER NOT NULL,
    project_id INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (project_id) REFERENCES projects(id)
);
```

### Таблица: projects

```sql
CREATE TABLE projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    created_by INTEGER NOT NULL,
    department VARCHAR(100) NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (created_by) REFERENCES users(id)
);
```

### Таблица: user_projects

```sql
CREATE TABLE user_projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    project_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, project_id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (project_id) REFERENCES projects(id)
);
```

## API Endpoints

### Аутентификация

- `POST /api/register` - Регистрация
- `POST /api/login` - Вход в систему
- `GET /api/auth/profile` - Профиль текущего пользователя
- `POST /api/auth/change-password` - Смена пароля

### Задачи

- `GET /api/tasks` - Получить список задач
- `POST /api/tasks` - Создать задачу (manager, admin)
- `PUT /api/tasks/:id` - Обновить задачу (manager, admin)
- `DELETE /api/tasks/:id` - Удалить задачу (manager, admin)

### Проекты

- `GET /api/projects` - Получить все проекты (manager, admin)
- `GET /api/projects/user` - Проекты текущего пользователя
- `POST /api/projects` - Создать проект (manager, admin)
- `PUT /api/projects/:id` - Обновить проект (manager, admin)
- `DELETE /api/projects/:id` - Удалить проект (admin)
- `POST /api/projects/assign` - Назначить пользователя на проект (manager, admin)
- `POST /api/projects/remove` - Удалить пользователя из проекта (manager, admin)
- `GET /api/projects/:id/users` - Получить пользователей проекта (manager, admin)

### Пользователи

- `GET /api/users` - Получить список пользователей
- `PUT /api/users/:id/role` - Изменить роль (admin)
- `PUT /api/users/:id/department` - Изменить отдел (admin)
- `POST /api/users/:id/reset-password` - Сбросить пароль (admin)
- `DELETE /api/users/:id` - Удалить пользователя (admin)

### Отчеты

- `GET /api/reports/my-tasks` - Экспорт своих задач
- `GET /api/reports/department-tasks` - Экспорт задач отдела (manager, admin)
- `GET /api/reports/all-tasks` - Экспорт всех задач (admin)

### Управление паролями

- `POST /api/password-reset-requests` - Создать заявку (публичный)
- `GET /api/password-reset-requests` - Получить заявки (admin)
- `POST /api/password-reset-requests/:id/process` - Обработать заявку (admin)
- `POST /api/password-reset-requests/:id/reject` - Отклонить заявку (admin)

### Active Directory

- `GET /api/ad-config/status` - Статус AD (публичный)
- `GET /api/ad-config` - Получить конфигурацию (admin)
- `POST /api/ad-config` - Сохранить конфигурацию (admin)
- `POST /api/ad-config/test` - Проверить подключение (admin)
- `POST /api/ad-config/sync` - Синхронизировать пользователей (admin)

### Резервная копия

- `GET /api/backup` - Скачать резервную копию (admin)
- `POST /api/restore` - Восстановить из резервной копии (admin)

## Разработка

### Запуск в режиме разработки

```bash
# Backend
cd backend
go run main.go

# Frontend (новый терминал)
cd frontend
npm start
```

### Добавление новой функции

1. **Backend:** Добавить обработчик в `handlers/`
2. **Backend:** Добавить модель в `models/models.go` (если требуется)
3. **Backend:** Добавить миграцию в `database/database.go` (если требуется)
4. **Frontend:** Создать компонент в `components/`
5. **Frontend:** Добавить API вызов в `utils/api.js`

### Тестирование

```bash
cd backend
go test ./...
```

## Логика приложения

### Аутентификация и авторизация

**Поток аутентификации:**

1. Пользователь вводит логин и пароль на странице Login
2. Frontend отправляет `POST /api/login` с учетными данными
3. Backend проверяет наличие пользователя в БД:
   - Если `is_ad_user = true` - пытается аутентифицировать через LDAP
   - Если `is_ad_user = false` - сравнивает хеши паролей (bcrypt)
4. При успехе генерируется JWT токен с информацией о пользователе:
   ```go
   claims := jwt.MapClaims{
       "user_id":     user.ID,
       "username":    user.Username,
       "role":        user.Role,
       "department":  user.Department,
       "exp":         time.Now().Add(24 * time.Hour).Unix(),
   }
   ```
5. Frontend сохраняет токен в localStorage и добавляет его в заголовок `Authorization: Bearer {token}` для всех последующих запросов
6. Backend проверяет токен в middleware для каждого защищенного эндпоинта

**Система ролей и прав:**

```
USER (простой сотрудник)
├─ Видит только свои задачи
├─ Может создавать/редактировать/удалять свои задачи
└─ Может экспортировать только свои задачи

MANAGER (менеджер)
├─ Видит все задачи своего отдела
├─ Может создавать/редактировать задачи в своем отделе
├─ Может добавлять сотрудников своего отдела на проекты
├─ Может создавать проекты для своего отдела
└─ Видит статистику своего отдела

ADMIN (администратор)
├─ Видит всё и может всё
├─ Управляет пользователями
├─ Конфигурирует систему
└─ Управляет AD/FreeIPA интеграцией
```

**Проверка прав (middleware):**

```go
// Только админ
func AdminOnly() gin.HandlerFunc {
    return func(c *gin.Context) {
        role := c.GetString("userRole")
        if role != "admin" {
            c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
            c.Abort()
            return
        }
        c.Next()
    }
}

// Менеджер или админ
func ManagerOrAdmin() gin.HandlerFunc {
    return func(c *gin.Context) {
        role := c.GetString("userRole")
        if role != "manager" && role != "admin" {
            c.JSON(http.StatusForbidden, gin.H{"error": "Manager or admin access required"})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

### Управление задачами

**Создание задачи:**

1. Менеджер/админ заполняет форму в TaskManager
2. Frontend отправляет `POST /api/tasks` с данными
3. Backend валидирует:
   - Пользователь (user_id) существует в БД
   - Если менеджер - пользователь должен быть из его отдела
   - Обязательные поля не пусты
4. Вставляет в БД: `INSERT INTO tasks (...) VALUES (...)`
5. Возвращает созданную задачу с ID

**Фильтрация при получении списка:**

```go
// Обычный пользователь видит только свои задачи
if userRole == "user" {
    rows = h.db.Query("SELECT * FROM tasks WHERE user_id = ?", userID)
}

// Менеджер видит задачи своего отдела
if userRole == "manager" {
    rows = h.db.Query(`
        SELECT t.* FROM tasks t
        JOIN users u ON t.user_id = u.id
        WHERE u.department = ?
    `, userDepartment)
}

// Админ видит всё
if userRole == "admin" {
    rows = h.db.Query("SELECT * FROM tasks")
}
```

**Обновление задачи:**

1. Проверяется, может ли пользователь редактировать:
   - Если user - только свои задачи
   - Если manager - задачи своего отдела
   - Если admin - любые
2. Обновляются все поля: `UPDATE tasks SET ... WHERE id = ?`
3. `updated_at` устанавливается автоматически (CURRENT_TIMESTAMP)

### Управление проектами

**Логика назначения на проект:**

1. Менеджер выбирает проект и сотрудника
2. Frontend отправляет `POST /api/projects/assign` с user_id и project_id
3. Backend проверяет:
   - Менеджер может назначать только из своего отдела
   - Админ может назначать любого
4. Проверка уникальности: `UNIQUE(user_id, project_id)` в таблице user_projects
5. Добавляет в БД: `INSERT INTO user_projects (user_id, project_id) VALUES (...)`

**Важная особенность:** Менеджер может видеть всех пользователей своего отдела (все роли), но управлять может только теми, кто находится в его отделе.

### Интеграция с LDAP/Active Directory

**Поток синхронизации:**

```
1. Администратор запускает синхронизацию
   ↓
2. Backend подключается к LDAP серверу
   (использует Bind DN и пароль для аутентификации)
   ↓
3. Выполняет LDAP поиск всех пользователей
   Base DN + User Search Base = полный путь поиска
   ↓
4. Для каждого пользователя из LDAP:
   - Извлекает атрибуты (username, department, email и т.д.)
   - Проверяет, существует ли в нашей БД
   - Если нет → INSERT (создаёт пользователя с is_ad_user=1)
   - Если есть → UPDATE (обновляет department и другие данные)
   ↓
5. Возвращает количество синхронизированных пользователей
```

**Аутентификация через LDAP:**

```go
func (m *LDAPManager) AuthenticateADUser(username, password string) (*UserInfo, error) {
    // 1. Подключается с сервисным аккаунтом
    conn := m.connect(bindDN, bindPassword)
    
    // 2. Ищет пользователя по username
    userDN := m.findUserDN(conn, username)
    
    // 3. Пытается подключиться с паролем пользователя
    userConn := m.connect(userDN, password)
    
    // 4. Если успешно - извлекает его атрибуты
    userInfo := m.getUserInfo(userConn, userDN)
    
    return userInfo, nil
}
```

### Экспорт в Excel

**Логика формирования отчета:**

```go
// 1. Получает данные из БД
rows := h.db.Query(`
    SELECT 
        ROW_NUMBER() OVER (ORDER BY u.id),
        u.username,
        p.name,
        t.weekly_info,
        t.planning,
        t.help_needed,
        t.hours_per_week,
        t.load_per_month
    FROM tasks t
    JOIN users u ON t.user_id = u.id
    LEFT JOIN projects p ON t.project_id = p.id
`)

// 2. Создает Excel файл с колонками
f := excelize.NewFile()
f.SetCellValue("Sheet1", "A1", "№")
f.SetCellValue("Sheet1", "B1", "ФИО")
// ... остальные колонки

// 3. Заполняет данные в таблицу
for row, task := range tasks {
    f.SetCellValue("Sheet1", fmt.Sprintf("A%d", row), task.RowNum)
    f.SetCellValue("Sheet1", fmt.Sprintf("B%d", row), task.Username)
    // ... остальные ячейки
}

// 4. Отправляет браузеру для скачивания
c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
c.Header("Content-Disposition", "attachment; filename=report.xlsx")
f.Write(c.Writer)
```

### Управление паролями

**Сброс пароля (только для обычных пользователей, не AD):**

```
Пользователь забыл пароль
    ↓
Отправляет заявку со своим username
    ↓
Админ видит заявку в системе
    ↓
Админ нажимает "Обработать"
    ↓
Система генерирует временный пароль (криптографически стойкий)
    ↓
Админ копирует пароль и передает пользователю
    ↓
Пользователь входит с временным паролем
    ↓
Система заставляет установить новый пароль
    ↓
password_reset_required = false
```

**Особенность:** Заявка может быть отклонена (status = 'rejected'), если это была попытка восстановления для AD пользователя, которому нужно сбросить пароль в самом AD.

### Frontend логика (React)

**Управление состоянием через Context:**

```javascript
// AuthContext хранит глобальное состояние
const [user, setUser] = useState(null);  // Текущий пользователь
const [loading, setLoading] = useState(true);

// При загрузке приложения проверяет localStorage на токен
useEffect(() => {
    const token = localStorage.getItem('token');
    const userData = localStorage.getItem('user');
    if (token && userData) {
        setUser(JSON.parse(userData));
    }
    setLoading(false);
}, []);

// Функция входа
const login = (userData, token) => {
    localStorage.setItem('token', token);
    localStorage.setItem('user', JSON.stringify(userData));
    setUser(userData);
};
```

**Автоматическое добавление токена к запросам:**

```javascript
// utils/api.js
const api = axios.create({
    baseURL: getApiBaseUrl(),
});

api.interceptors.request.use((config) => {
    const token = localStorage.getItem('token');
    if (token) {
        config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
});
```

**Условный рендеринг по ролям:**

```javascript
// Например, в TaskManager
if (user?.role === 'manager' || user?.role === 'admin') {
    // Показать форму создания задачи
}

if (user?.role === 'admin') {
    // Показать кнопку "Синхронизировать AD"
}
```

## Примеры кода

### Backend: Создание задачи с проверкой прав

```go
func (h *TaskHandler) CreateTask(c *gin.Context) {
    userRole := c.GetString("userRole")
    userID := c.GetInt("userID")
    userDepartment := c.GetString("userDepartment")

    // 1. Проверка прав
    if userRole != "admin" && userRole != "manager" {
        c.JSON(http.StatusForbidden, gin.H{"error": "Недостаточно прав"})
        return
    }

    // 2. Парсинг запроса
    var task models.Task
    if err := c.ShouldBindJSON(&task); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // 3. Проверка, что менеджер создает задачу для своего отдела
    if userRole == "manager" {
        var targetDept sql.NullString
        err := h.db.QueryRow(
            "SELECT department FROM users WHERE id = ?", 
            task.UserID,
        ).Scan(&targetDept)
        
        if err != nil || !targetDept.Valid || targetDept.String != userDepartment {
            c.JSON(http.StatusForbidden, gin.H{
                "error": "Вы можете создавать задачи только для своего отдела",
            })
            return
        }
    }

    // 4. Вставка в БД
    result, err := h.db.Exec(`
        INSERT INTO tasks (
            title, description, progress, hours_per_week,
            load_per_month, weekly_info, planning, help_needed,
            user_id, project_id, created_at, updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `,
        task.Title, task.Description, task.Progress,
        task.HoursPerWeek, task.LoadPerMonth,
        task.WeeklyInfo, task.Planning, task.HelpNeeded,
        task.UserID, task.ProjectID,
        time.Now(), time.Now(),
    )

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // 5. Получение ID и отправка ответа
    id, _ := result.LastInsertId()
    task.ID = int(id)
    task.CreatedAt = time.Now()
    
    c.JSON(http.StatusCreated, task)
}
```

### Frontend: Загрузка и фильтрация пользователей

```javascript
const fetchUsers = async () => {
    try {
        const response = await api.get('/api/users');
        if (response.data) {
            let userList = response.data;
            
            // Только для менеджера - фильтруем по отделу
            if (user?.role === 'manager') {
                userList = userList.filter(
                    u => u.department === user?.department
                );
            }
            // Для админа - всех
            
            setUsers(userList);
        }
    } catch (error) {
        console.error('Error fetching users:', error);
    }
};
```

## Поток данных в критических местах

### Создание и назначение на проект

```
Frontend (ProjectManagement.js)
    ↓
Пользователь выбирает проект и сотрудника
    ↓
POST /api/projects/assign {user_id, project_id}
    ↓
Backend (projects.go: AssignProjectToUser)
    ↓
1. Проверка прав (менеджер ↔ свой отдел, админ ↔ все)
2. INSERT INTO user_projects (user_id, project_id)
3. Возвращает 201 Created
    ↓
Frontend получает успех
    ↓
Обновляет список назначенных пользователей
    ↓
Пользователь видит сотрудника в проекте
```

## Технологический стек

- **Frontend:** React 18, Axios, Context API, CSS3
- **Backend:** Go 1.20+, Gin Framework, SQLite3, JWT, excelize, LDAP
- **DevOps:** Docker, Docker Compose, nginx
- **Database:** SQLite3 (встроенная, не требует отдельного сервера)
- **Security:** bcrypt (пароли), JWT (токены), crypto/aes (шифрование)

---

## Решение проблем (Troubleshooting)

### Docker проблемы

**Проблема:** Порт уже занят
```bash
docker-compose down
# Отредактируйте docker-compose.yml и измените порт
```

**Проблема:** `ENCRYPTION_KEY environment variable is not set`
```bash
cd backend
python3 scripts/generate_encryption_key.py
```

### Frontend проблемы

**Проблема:** CORS ошибка
```
1. Убедитесь, что backend запущен на http://localhost:8080
2. Проверьте baseURL в компонентах frontend
3. Проверьте firewall
```

**Проблема:** Белый экран
```bash
docker-compose logs -f frontend
docker-compose restart
```

### Backend проблемы

**Проблема:** `database is locked`
```bash
docker-compose restart backend
docker-compose down -v
docker-compose up -d
```

**Проблема:** LDAP/AD не подключается
```
1. Проверьте URL LDAP сервера (ldap://host:389)
2. Проверьте Bind DN и пароль
3. Проверьте доступность сервера из контейнера
```

### Авторизация

**Проблема:** Забыли пароль admin
```bash
docker-compose down -v
docker-compose up -d
# Система создаст admin с паролем main12!@
```

**Проблема:** JWT токен истек
```bash
# Удалите 'token' и 'user' из localStorage
# Заново войдите в систему
```

---

## Поддержка и развитие

### Источники информации

- **Логи:** `docker-compose logs -f backend` / `docker-compose logs -f frontend`
- **База данных:** `./backend/data/tasks.db`
- **Конфигурация:** `./backend/.env`

---

**Последнее обновление документации:** 23 января 2026 г.
