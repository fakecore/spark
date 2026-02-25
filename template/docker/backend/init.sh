#!/bin/sh
set -e

echo "Waiting for MySQL to be ready..."
while ! nc -z db 3306; do
  sleep 1
done

echo "MySQL is ready, initializing database..."
# 在这里添加您的数据库初始化命令
# 例如：
# mysql -h db -u myuser -pmypassword mydb < /app/init.sql

echo "Database initialized. Starting the application..."
./main