@echo off
setlocal enabledelayedexpansion

echo ======================================
echo   Build Embedded Executable (Windows)
echo ======================================
echo.

REM Project root directory
set PROJECT_ROOT=%~dp0..
cd %PROJECT_ROOT%

REM Step 1: Clean frontend dist
echo [1/4] Cleaning frontend build output...
if exist xingran-react-frontend\dist (
    rmdir /s /q xingran-react-frontend\dist
)
echo Frontend dist cleaned
echo.

REM Step 2: Build frontend
echo [2/5] Building frontend React app...
cd xingran-react-frontend
call npm run build
if errorlevel 1 (
    echo Frontend build FAILED!
    exit /b 1
)
cd ..
echo Frontend build completed
echo.

REM Step 3: Copy frontend to embed directory
echo [3/5] Copying frontend to embed directory...
if exist internal\server\xingran-react-frontend (
    rmdir /s /q internal\server\xingran-react-frontend
)
mkdir internal\server\xingran-react-frontend
xcopy /e /i /y xingran-react-frontend\dist internal\server\xingran-react-frontend\dist
echo Frontend files copied to embed directory
echo.

REM Step 4: Build Go backend
echo [4/5] Building Go backend (Windows)...
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -tags=embed -ldflags="-s -w" -o xingran-backend-embedded.exe ./cmd/main.go
if errorlevel 1 (
    echo Go build FAILED!
    exit /b 1
)
echo Go build completed
echo.

REM Step 5: Complete
echo [5/5] Build completed!
echo.
echo ======================================
echo   Output File
echo ======================================
echo   xingran-backend-embedded.exe
echo.
echo Start server:
echo   .\xingran-backend-embedded.exe
echo.

endlocal