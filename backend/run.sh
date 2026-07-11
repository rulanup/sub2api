#!/bin/bash
export DATABASE_USER=sub2api
export DATABASE_PASSWORD=sub2api123
export DATABASE_HOST=localhost
export DATABASE_PORT=5432
export DATABASE_SSLMODE=disable
export DATABASE_DBNAME=sub2api
export REDIS_HOST=localhost
export REDIS_PORT=6379
export SERVER_HOST=0.0.0.0
export SERVER_PORT=8080
export SERVER_MODE=debug
export AUTO_SETUP=true
export ADMIN_EMAIL=admin@test.com
export ADMIN_PASSWORD=admin123
export JWT_SECRET=abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890
export TOTP_ENCRYPTION_KEY=$(openssl rand -hex 32)
cd /home/guili/sub2api/backend
exec ./bin/server
