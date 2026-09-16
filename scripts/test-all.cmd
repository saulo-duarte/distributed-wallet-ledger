@echo off
setlocal EnableExtensions

set "PROJECT_ROOT=%~dp0.."
for %%I in ("%PROJECT_ROOT%") do set "PROJECT_ROOT=%%~fI"
set "CACHE_DIR=%PROJECT_ROOT%\.cache"
set "GOCACHE=%CACHE_DIR%\go"
set "COVERAGE_FILE=%PROJECT_ROOT%\coverage.out"

if not exist "%GOCACHE%" mkdir "%GOCACHE%"

pushd "%PROJECT_ROOT%"

echo [1/5] Checking formatting...
gofmt -l cmd internal > "%CACHE_DIR%\gofmt-files.txt"
for %%A in ("%CACHE_DIR%\gofmt-files.txt") do if %%~zA GTR 0 goto formatting_failed

echo [2/5] Running tests...
go test -count=1 ./...
if errorlevel 1 goto validation_failed

echo [3/5] Generating coverage profile...
go test -count=1 -coverprofile="%COVERAGE_FILE%" ./...
if errorlevel 1 goto validation_failed
go tool cover -func="%COVERAGE_FILE%"
if errorlevel 1 goto validation_failed

echo [4/5] Running go vet...
go vet ./...
if errorlevel 1 goto validation_failed

echo [5/5] Building all packages...
go build ./...
if errorlevel 1 goto validation_failed

if /I "%RUN_FUZZ%"=="1" (
    echo [optional] Running fuzz tests...
    call "%PROJECT_ROOT%\scripts\test-fuzz.cmd"
    if errorlevel 1 goto validation_failed
)

if /I "%RUN_MUTATION%"=="1" (
    echo [optional] Running mutation tests...
    call "%PROJECT_ROOT%\scripts\test-mutation.cmd"
    if errorlevel 1 goto validation_failed
)

echo.
echo All requested validations passed.
popd
exit /b 0

:formatting_failed
echo Formatting check failed. Run: gofmt -w ./cmd ./internal
type "%CACHE_DIR%\gofmt-files.txt"
goto validation_failed

:validation_failed
echo Validation failed.
popd
exit /b 1
