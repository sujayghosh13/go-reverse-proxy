@echo off
title Launching Backends and Proxy in Tabs...
cd /d "%~dp0"

where wt >nul 2>nul
if %errorlevel% neq 0 goto :NO_WT

echo Launching all 3 backends, Reverse Proxy, and Terminal in 1 Windows Terminal window (tabs)...
start wt -d "%CD%" cmd /k "title Backend 1 && go run backends\backend1.go" ; new-tab -d "%CD%" cmd /k "title Backend 2 && go run backends\backend2.go" ; new-tab -d "%CD%" cmd /k "title Backend 3 && go run backends\backend3.go" ; new-tab -d "%CD%" cmd /k "title Reverse Proxy (Port 8080) && go run main.go" ; new-tab -d "%CD%" cmd /k "title Terminal (d:\reverse-proxy)"
exit /b 0

:NO_WT
echo Windows Terminal (wt) not found. Opening in separate windows...
for %%f in (backends\*.go) do (
    echo [STARTING] %%~nf - %%f
    start "%%~nf" cmd /k "title %%~nf && go run %%f"
)
timeout /t 2 /nobreak > nul
start "Reverse Proxy" cmd /k "title Reverse Proxy && go run main.go"
start "Terminal" cmd /k "cd /d %CD%"
exit /b 0
