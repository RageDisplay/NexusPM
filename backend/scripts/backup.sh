#!/bin/bash

# Пути и настройки
BACKUP_SOURCE="./backend/data/tasks.db"
BACKUP_DIR="./auto_backups"
MAX_BACKUPS=5

mkdir -p "$BACKUP_DIR"

TIMESTAMP=$(date +"%Y-%m-%d_%H-%M-%S")
BACKUP_FILE="$BACKUP_DIR/tasks_${TIMESTAMP}.db"

echo "Creating backup: $BACKUP_FILE"
cp "$BACKUP_SOURCE" "$BACKUP_FILE"

if [ $? -eq 0 ]; then
    echo "Бекап создан"
else
    echo "Error: Бекап не удалось создать"
    exit 1
fi

echo "Очистка..."
cd "$BACKUP_DIR" || exit

ls -1t tasks_*.db 2>/dev/null | tail -n +$(($MAX_BACKUPS + 1)) | while read -r old_backup; do
    echo "Removing old backup: $old_backup"
    rm -- "$old_backup"
done

echo "Бекап завершён. Текущие:"
ls -1t tasks_*.db 2>/dev/null | head -$MAX_BACKUPS