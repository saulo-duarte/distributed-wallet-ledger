@echo off
setlocal EnableExtensions

set "PROJECT_ROOT=%~dp0.."
for %%I in ("%PROJECT_ROOT%") do set "PROJECT_ROOT=%%~fI"
set "CACHE_DIR=%PROJECT_ROOT%\.cache"
set "GOCACHE=%CACHE_DIR%\go"
set "MUTATION_OUTPUT=%CACHE_DIR%\gremlins-domain.json"
if not exist "%CACHE_DIR%" mkdir "%CACHE_DIR%"
if not exist "%GOCACHE%" mkdir "%GOCACHE%"

set "GREMLINS=gremlins"
where gremlins >nul 2>&1
if errorlevel 1 for /f "delims=" %%A in ('go env GOPATH') do set "GREMLINS=%%A\bin\gremlins.exe"
if not exist "%GREMLINS%" if /I not "%GREMLINS%"=="gremlins" (
    echo Gremlins was not found on PATH.
    echo Install it from https://github.com/go-gremlins/gremlins and run this script again.
    exit /b 2
)

if not defined MUTATION_EFFICACY_THRESHOLD set "MUTATION_EFFICACY_THRESHOLD=80"
if not defined MUTATION_COVERAGE_THRESHOLD set "MUTATION_COVERAGE_THRESHOLD=80"
if not defined MUTATION_WORKERS set "MUTATION_WORKERS=1"
if not defined MUTATION_TIMEOUT_COEFFICIENT set "MUTATION_TIMEOUT_COEFFICIENT=3"

pushd "%PROJECT_ROOT%\internal\ledger\domain"
"%GREMLINS%" unleash --output="%MUTATION_OUTPUT%" --workers=%MUTATION_WORKERS% --timeout-coefficient=%MUTATION_TIMEOUT_COEFFICIENT% --threshold-efficacy=%MUTATION_EFFICACY_THRESHOLD% --threshold-mcover=%MUTATION_COVERAGE_THRESHOLD%
if errorlevel 1 goto mutation_failed

findstr /c:"TIMED_OUT" "%MUTATION_OUTPUT%" >nul
if not errorlevel 1 (
    echo Warning: mutation testing produced timed out mutants. Review the JSON report.
)

popd
echo Mutation testing passed.
echo Report: %MUTATION_OUTPUT%
exit /b 0

:mutation_failed
echo Mutation testing failed or did not meet the configured thresholds.
popd
exit /b 1
