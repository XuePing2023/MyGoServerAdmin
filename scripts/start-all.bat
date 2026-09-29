@echo off
rem ServerAdmin one-click start: MongoDB + admin server (Windows)
rem Usage: double-click, or run scripts\start-all.bat in cmd

echo [1/2] Starting MongoDB (127.0.0.1:27017) ...
start "ServerAdmin-MongoDB" /min "D:\Programs\mongodb-win32-x86_64-windows-7.0.14\bin\mongod.exe" --dbpath "D:\Temp\mongo-data" --port 27017 --bind_ip 127.0.0.1 --logpath "D:\Temp\mongo-data\mongod.log"

timeout /t 4 /nobreak >nul

echo [2/2] Starting ServerAdmin (http://localhost:8080) ...
cd /d "%~dp0.."
start "ServerAdmin-App" /min bin\serveradmin.exe

echo Done! Open http://localhost:8080 (default account: admin / admin123)
