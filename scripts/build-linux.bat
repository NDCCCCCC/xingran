@echo off
setlocal enabledelayedexpansion

REM ============================================================
REM  Usage:
REM    scripts\build-linux.bat        -> base=/xingran/  (default)
REM        Tencent Cloud nginx sub-path deployment:
REM          location /xingran/ { proxy_pass http://127.0.0.1:9000/; }
REM        nginx strips the /xingran/ prefix, so frontend asset URLs
REM        and API URLs MUST carry that prefix.
REM
REM    scripts\build-linux.bat root  -> base=/
REM        Direct :9000 deployment (no nginx sub-path):
REM        browser opens http://<host>:9000/
REM        assets and API calls carry no prefix.
REM        (the literal "/" also works, but "root" is safer: MSYS in
REM         Git Bash rewrites a lone "/" argument into an absolute path)
REM
REM  The two deployments need different bases, so base MUST be a
REM  build-time argument and must NOT be hardcoded in .env.production
REM  (.env.production keeps /xingran/ for CI's plain `npm run build`).
REM
REM  Notes:
REM  - base is exported here and picked up by npm; in vite.config.ts
REM    process.env.VITE_BASE wins over the .env file value (vite's
REM    loadEnv also overwrites same-named VITE_* vars from process.env).
REM  - Do NOT export VITE_BASE from Git Bash: MSYS rewrites values that
REM    start with "/" into absolute paths (observed: it became the bash
REM    install dir). Running this script (cmd set) is unaffected.
REM  - Keep this file ASCII-only: cmd decodes .bat with the OEM code
REM    page, so UTF-8 Chinese comments get misparsed into commands.
REM ============================================================
set "VITE_BASE=%~1"
if "%VITE_BASE%"=="" set "VITE_BASE=/xingran/"
if /i "%VITE_BASE%"=="root" set "VITE_BASE=/"

REM Derive API prefix from base: /xingran/ -> /xingran/api/v1 ; / -> /api/v1
set "BASE_NOSLASH=%VITE_BASE%"
if "%BASE_NOSLASH:~-1%"=="/" set "BASE_NOSLASH=%BASE_NOSLASH:~0,-1%"
set "VITE_API_BASE_URL=%BASE_NOSLASH%/api/v1"

REM Distinct artifact name per base so the two deployments never mix up
set "OUT_DIR=out"
set "OUT_FILE=%OUT_DIR%\xingran-backend-embedded-linux"
if not "%VITE_BASE%"=="/xingran/" set "OUT_FILE=%OUT_DIR%\xingran-backend-embedded-linux-root"

echo ======================================
echo   Build Embedded Executable (Linux)
echo ======================================
echo   Frontend base : %VITE_BASE%
echo   API base      : %VITE_API_BASE_URL%
echo   Output file   : %OUT_FILE%
echo.

REM Project root directory
set PROJECT_ROOT=%~dp0..
cd %PROJECT_ROOT%

REM Step 1: Clean frontend dist
echo [1/5] Cleaning frontend build output...
if exist xingran-react-frontend\dist (
    rmdir /s /q xingran-react-frontend\dist
)
echo Frontend dist cleaned
echo.

REM Step 2: Check frontend dependencies
echo [2/5] Checking frontend dependencies...
cd xingran-react-frontend
if not exist node_modules (
    echo Installing frontend dependencies...
    call npm install
)
cd ..
echo.

REM Step 3: Build frontend
echo [3/5] Building frontend React app...
cd xingran-react-frontend
call npm run build
if errorlevel 1 (
    echo Frontend build FAILED!
    exit /b 1
)
cd ..
echo Frontend build completed
echo.

REM Step 4: Copy frontend to embed directory
echo [4/6] Copying frontend to embed directory...
if exist internal\server\xingran-react-frontend (
    rmdir /s /q internal\server\xingran-react-frontend
)
mkdir internal\server\xingran-react-frontend
xcopy /e /i /y xingran-react-frontend\dist internal\server\xingran-react-frontend\dist
echo Frontend files copied to embed directory
echo.

REM Step 5: Build Go backend (Linux)
echo [5/6] Building Go backend (Linux)...
if not exist "%OUT_DIR%" mkdir "%OUT_DIR%"
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -tags=embed -ldflags="-s -w" -o %OUT_FILE% ./cmd/main.go
if errorlevel 1 (
    echo Go build FAILED!
    exit /b 1
)
echo Go build completed
echo.

REM Step 6: Complete
echo [6/6] Build completed!
echo.
echo ======================================
echo   Output File
echo ======================================
echo   %OUT_FILE%
echo.
echo Upload to Linux server and run:
echo   chmod +x %OUT_FILE%
echo   ./%OUT_FILE%
echo.
echo NOTE: artifacts always land in out/ (gitignored); see AGENTS.md.

endlocal
