#!/bin/bash

# Пути и настройки
BACKUP_DIR="./auto_backups"
TARGET_DIR="./backend/data"
TARGET_FILE="$TARGET_DIR/tasks.db"

if [ ! -d "$BACKUP_DIR" ]; then
    echo "Error: Backup directory '$BACKUP_DIR' not found!"
    exit 1
fi

backups=($(ls -1t "$BACKUP_DIR"/tasks_*.db 2>/dev/null))

if [ ${#backups[@]} -eq 0 ]; then
    echo "No backups found in $BACKUP_DIR"
    exit 1
fi

echo "Доступные бекапы:"
echo "------------------"
for i in "${!backups[@]}"; do
    filename=$(basename "${backups[i]}")
    size=$(du -h "${backups[i]}" | cut -f1)
    date=$(stat -c %y "${backups[i]}" | cut -d'.' -f1)
    echo "$((i+1)). $filename (Size: $size, Modified: $date)"
done
echo "------------------"

read -p "Выберите бекап из списка (1-${#backups[@]}): " choice

if ! [[ "$choice" =~ ^[0-9]+$ ]] || [ "$choice" -lt 1 ] || [ "$choice" -gt ${#backups[@]} ]; then
    echo "Invalid selection!"
    exit 1
fi

selected_backup="${backups[$((choice-1))]}"

# Создаем резервную копию текущей базы (если существует)
if [ -f "$TARGET_FILE" ]; then
    backup_current="$BACKUP_DIR/current_before_restore_$(date +"%Y-%m-%d_%H-%M-%S").db"
    echo "Бекап текущей конфигурации при наличии: $backup_current"
    cp "$TARGET_FILE" "$backup_current"
fi

# Восстанавливаем выбранный бэкап
echo "Восстановление из: $(basename "$selected_backup")"
cp "$selected_backup" "$TARGET_FILE"

# Проверяем успешность восстановления
if [ $? -eq 0 ]; then
    echo "База данных восстановлена!"
    
    # Показываем информацию о восстановленном файле
    restored_size=$(du -h "$TARGET_FILE" | cut -f1)
    echo "Восстановлено из: $TARGET_FILE (Size: $restored_size)"
else
    echo "Error: Бекап не прошёл"
    
    # Восстанавливаем оригинал из резервной копии, если она создавалась
    if [ -n "$backup_current" ] && [ -f "$backup_current" ]; then
        echo "Восстановление к исходному виду..."
        cp "$backup_current" "$TARGET_FILE"
    fi
    exit 1
fi