@echo off
setlocal EnableExtensions

set "PROJECT_ROOT=%~dp0.."
for %%I in ("%PROJECT_ROOT%") do set "PROJECT_ROOT=%%~fI"
set "CACHE_DIR=%PROJECT_ROOT%\.cache"
set "GOCACHE=%CACHE_DIR%\go"
if not exist "%GOCACHE%" mkdir "%GOCACHE%"

for /f "delims=" %%A in ('go env GOARCH') do set "GOARCH=%%A"
if /I not "%GOARCH%"=="amd64" if /I not "%GOARCH%"=="arm64" (
    echo Fuzzing requires an AMD64 or ARM64 Go environment. Current architecture: %GOARCH%
    exit /b 2
)

if not defined FUZZ_TIME set "FUZZ_TIME=30s"

pushd "%PROJECT_ROOT%"
echo Running Currency fuzzing for %FUZZ_TIME%...
go test -fuzz=FuzzNewCurrency -fuzztime=%FUZZ_TIME% ./internal/ledger/domain
if errorlevel 1 goto fuzz_failed

echo Running ID fuzzing for %FUZZ_TIME%...
go test -fuzz=FuzzNewIDs -fuzztime=%FUZZ_TIME% ./internal/ledger/domain
if errorlevel 1 goto fuzz_failed

echo Running Money constructor fuzzing for %FUZZ_TIME%...
go test -fuzz=FuzzNewMoney -fuzztime=%FUZZ_TIME% ./internal/ledger/domain
if errorlevel 1 goto fuzz_failed

echo Running Money addition fuzzing for %FUZZ_TIME%...
go test -fuzz=FuzzMoneyAdd -fuzztime=%FUZZ_TIME% ./internal/ledger/domain
if errorlevel 1 goto fuzz_failed

echo Running JournalEntry balance fuzzing for %FUZZ_TIME%...
go test -fuzz=FuzzNewJournalEntryBalance -fuzztime=%FUZZ_TIME% ./internal/ledger/domain
if errorlevel 1 goto fuzz_failed

echo Running JournalEntry currency fuzzing for %FUZZ_TIME%...
go test -fuzz=FuzzNewJournalEntryCurrencyConsistency -fuzztime=%FUZZ_TIME% ./internal/ledger/domain
if errorlevel 1 goto fuzz_failed

popd
echo Fuzz tests passed.
exit /b 0

:fuzz_failed
echo Fuzz testing failed.
popd
exit /b 1
