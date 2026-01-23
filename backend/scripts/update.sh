#!/bin/bash
set -e

echo "Обновление Task Management System..."

docker-compose down

docker image prune -f

docker-compose pull

docker-compose up -d

echo "Обновление завершено"
echo "Приложение доступно по: http://localhost:80"