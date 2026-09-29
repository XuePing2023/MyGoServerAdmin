@echo off
rem Stop ServerAdmin and MongoDB
taskkill /f /im serveradmin.exe 2>nul
taskkill /f /im mongod.exe 2>nul
echo Stopped.
