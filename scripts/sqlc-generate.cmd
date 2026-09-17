@echo off
setlocal

if not exist .tools mkdir .tools
if not exist .cache\sqlc mkdir .cache\sqlc

if not exist .tools\sqlc.exe (
    set "GOBIN=%CD%\.tools"
    set "GOCACHE=%CD%\.cache\sqlc"
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0
    if errorlevel 1 exit /b %ERRORLEVEL%
)

.tools\sqlc.exe generate
exit /b %ERRORLEVEL%
