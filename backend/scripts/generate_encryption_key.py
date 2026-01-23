#!/usr/bin/env python3
import os
import secrets
import base64
import sys

def generate_encryption_key():
    """Генерирует 256-битный ключ (32 байта) для AES"""
    key = secrets.token_bytes(32)
    return base64.b64encode(key).decode('utf-8')

def create_env_file(env_file='.env'):
    """Создаёт .env файл с ключом шифрования"""
    if os.path.exists(env_file):
        response = input(f"Файл {env_file} уже существует. Перезаписать? (y/n): ").strip().lower()
        if response != 'y':
            print("Отменено")
            return False
    
    key = generate_encryption_key()
    
    with open(env_file, 'w') as f:
        f.write(f"ENCRYPTION_KEY={key}\n")
    
    print(f"✓ Файл {env_file} создан")
    print(f"✓ Ключ шифрования: {key}")
    return True

if __name__ == '__main__':
    env_file = sys.argv[1] if len(sys.argv) > 1 else '.env'
    create_env_file(env_file)
