@echo off
setlocal

if "%POMELO_ORBIT_APP__ENV%"=="" set "POMELO_ORBIT_APP__ENV=release"
cd /d "%~dp0"
pomelo-orbit.exe %*
exit /b %ERRORLEVEL%
