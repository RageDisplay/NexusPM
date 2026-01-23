#!/bin/bash
set -e

# Проверка Docker
if ! command -v docker &> /dev/null; then
    echo "Ошибка: Docker не установлен"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo "Ошибка: Docker Compose не установлен"
    exit 1
fi

echo "Docker и Docker Compose проверены"

# Создаем необходимые директории
mkdir -p data/backups

# Скачиваем образы
echo "Загрузка образов...."
docker-compose pull

# Запускаем приложение
echo "Запуск приложения..."
docker-compose up -d

echo "Ожидание запуска сервисов..."
sleep 10

# Проверяем статус
if docker-compose ps | grep -q "Up"; then
    echo "Приложение успешно запущено"
    echo ""
    echo "Доступ к приложению: http://localhost:80"
    echo "API сервер: http://localhost:8080"
    echo ""
    echo "Данные для входа администратора:"
    echo "Логин: admin"
    echo "Пароль: main12!@"
    echo ""
    echo "Команды управления:"
    echo "Просмотр логов: docker-compose logs -f"
    echo "Остановка: docker-compose down"
    echo "Перезапуск: docker-compose restart"
    echo "Обновление: ./update.sh"
else
    echo "Ошибка при запуске приложения"
    docker-compose logs
    exit 1
fi