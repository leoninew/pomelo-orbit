@echo off
setlocal EnableExtensions

set "POMELO_ORBIT_APP__ENV=release"
set "POMELO_ORBIT_DIR=%~dp0"
set "POMELO_ORBIT_EXE=%POMELO_ORBIT_DIR%pomelo-orbit.exe"
set "POMELO_ORBIT_TOOL_BIN=%USERPROFILE%\.local\bin"

if exist "%POMELO_ORBIT_TOOL_BIN%" set "PATH=%POMELO_ORBIT_TOOL_BIN%;%PATH%"

if "%~1"=="" goto :run
if /I "%~1"=="start" goto :run_without_control_arg
if /I "%~1"=="status" goto :status
if /I "%~1"=="stop" goto :stop
if /I "%~1"=="restart" goto :restart

:run
cd /d "%POMELO_ORBIT_DIR%" || exit /b 1
"%POMELO_ORBIT_EXE%" %*
exit /b %ERRORLEVEL%

:status
call :find_running_pids
if not defined POMELO_ORBIT_PIDS (
    echo pomelo-orbit is not running from "%POMELO_ORBIT_EXE%".
    exit /b 3
)
echo pomelo-orbit is running from "%POMELO_ORBIT_EXE%" with PID(s): %POMELO_ORBIT_PIDS%
exit /b 0

:stop
call :stop_running
exit /b %ERRORLEVEL%

:restart
call :stop_running
if errorlevel 1 exit /b %ERRORLEVEL%
goto :run_without_control_arg

:run_without_control_arg
cd /d "%POMELO_ORBIT_DIR%" || exit /b 1
"%POMELO_ORBIT_EXE%"
exit /b %ERRORLEVEL%

:stop_running
call :find_running_pids
if not defined POMELO_ORBIT_PIDS (
    echo pomelo-orbit is not running from "%POMELO_ORBIT_EXE%".
    exit /b 0
)

echo Stopping pomelo-orbit PID(s): %POMELO_ORBIT_PIDS%
for %%P in (%POMELO_ORBIT_PIDS%) do taskkill /PID %%P /T >nul 2>&1
timeout /t 3 /nobreak >nul

call :find_running_pids
if defined POMELO_ORBIT_PIDS (
    echo Forcing pomelo-orbit PID(s): %POMELO_ORBIT_PIDS%
    for %%P in (%POMELO_ORBIT_PIDS%) do taskkill /PID %%P /T /F >nul 2>&1
)

call :find_running_pids
if defined POMELO_ORBIT_PIDS (
    echo Failed to stop pomelo-orbit PID(s): %POMELO_ORBIT_PIDS%
    exit /b 1
)

echo pomelo-orbit stopped.
exit /b 0

:find_running_pids
set "POMELO_ORBIT_PIDS="
for /f %%P in ('powershell.exe -NoProfile -Command "$target = [System.IO.Path]::GetFullPath($env:POMELO_ORBIT_EXE); foreach ($process in Get-CimInstance Win32_Process) { if ($process.Name -ieq 'pomelo-orbit.exe' -and $process.ExecutablePath -and [System.IO.Path]::GetFullPath($process.ExecutablePath) -ieq $target) { $process.ProcessId } }"') do (
    if defined POMELO_ORBIT_PIDS (
        set "POMELO_ORBIT_PIDS=%POMELO_ORBIT_PIDS% %%P"
    ) else (
        set "POMELO_ORBIT_PIDS=%%P"
    )
)
exit /b 0
