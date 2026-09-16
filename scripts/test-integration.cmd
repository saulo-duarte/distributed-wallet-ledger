@echo off
setlocal EnableExtensions

set "PROJECT_ROOT=%~dp0.."
for %%I in ("%PROJECT_ROOT%") do set "PROJECT_ROOT=%%~fI"
set "CACHE_DIR=%PROJECT_ROOT%\.cache"
set "GOCACHE=%CACHE_DIR%\go"
if not exist "%GOCACHE%" mkdir "%GOCACHE%"

if exist "%PROJECT_ROOT%\.env" (
    for /f "usebackq eol=# tokens=1,* delims==" %%A in ("%PROJECT_ROOT%\.env") do (
        if not defined %%A set "%%A=%%B"
    )
)

if not defined DATABASE_URL set "DATABASE_URL=postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable"
if not defined MIGRATIONS_PATH set "MIGRATIONS_PATH=migrations"
if not defined MIGRATE set "MIGRATE=migrate"

where docker >nul 2>&1
if errorlevel 1 (
    echo Docker was not found on PATH.
    exit /b 2
)

if /I "%MIGRATE%"=="migrate" (
    where migrate >nul 2>&1
    if errorlevel 1 (
        echo The migrate CLI was not found on PATH.
        echo Install golang-migrate or set MIGRATE to its executable path.
        exit /b 2
    )
)

pushd "%PROJECT_ROOT%"

echo Starting local PostgreSQL...
docker compose up -d --wait postgres
if errorlevel 1 goto integration_failed

echo Applying database migrations...
"%MIGRATE%" -path "%MIGRATIONS_PATH%" -database "%DATABASE_URL%" up
if errorlevel 1 goto integration_failed

echo Running integration tests...
go test -tags=integration ./...
if errorlevel 1 goto integration_failed

echo Integration tests passed.
popd
exit /b 0

:integration_failed
echo Integration test setup or execution failed.
popd
exit /b 1
