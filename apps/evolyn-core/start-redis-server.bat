@echo off
title Redis Server

cd /d D:\dev\Redis-x64-3.2.100

echo Starting Redis Server...
redis-server.exe redis.windows.conf

pause
