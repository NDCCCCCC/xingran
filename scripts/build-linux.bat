@echo off
setlocal enabledelayedexpansion

echo ======================================
echo   Build Embedded Executable (Linux)
echo ======================================
echo.

REM Project root directory
set PROJECT_ROOT=%~dp0..
cd %PROJECT_ROOT%

REM Step 1: Clean frontend dist
echo [1/5] Cleaning frontend build output...
if exist ruoyi-react-frontend\dist (
    rmdir /s /q ruoyi-react-frontend\dist
)
echo Frontend dist cleaned
echo.

REM Step 2: Check frontend dependencies
echo [2/5] Checking frontend dependencies...
cd ruoyi-react-frontend
if not exist node_modules (
    echo Installing frontend dependencies...
    call npm install
)
cd ..
echo.

REM Step 3: Build frontend
echo [3/5] Building frontend React app...
cd ruoyi-react-frontend
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
if exist internal\server\ruoyi-react-frontend (
    rmdir /s /q internal\server\ruoyi-react-frontend
)
mkdir internal\server\ruoyi-react-frontend
xcopy /e /i /y ruoyi-react-frontend\dist internal\server\ruoyi-react-frontend\dist
echo Frontend files copied to embed directory
echo.

REM Step 5: Build Go backend (Linux)
echo [5/6] Building Go backend (Linux)...
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -tags=embed -ldflags="-s -w" -o ruoyi-backend-embedded-linux ./cmd/main.go
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
echo   ruoyi-backend-embedded-linux
echo.
echo Upload to Linux server and run:
echo   chmod +x ruoyi-backend-embedded-linux
echo   ./ruoyi-backend-embedded-linux
echo.

endlocal
