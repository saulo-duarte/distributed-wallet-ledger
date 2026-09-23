@echo off
setlocal EnableExtensions

set "PROJECT_ROOT=%~dp0.."
for %%I in ("%PROJECT_ROOT%") do set "PROJECT_ROOT=%%~fI"

where docker >nul 2>&1
if errorlevel 1 (
    echo Docker was not found on PATH.
    exit /b 2
)

where terraform >nul 2>&1
if errorlevel 1 (
    echo Terraform was not found on PATH.
    exit /b 2
)

pushd "%PROJECT_ROOT%"

echo Starting PostgreSQL and Ministack...
docker compose up -d --wait postgres ministack
if errorlevel 1 goto infra_failed

echo Provisioning local DynamoDB, SNS, SQS, DLQ, and SSM resources...
terraform -chdir=terraform/local init -backend=false
if errorlevel 1 goto infra_failed

terraform -chdir=terraform/local apply -auto-approve
if errorlevel 1 goto infra_failed

echo Local infrastructure is ready.
popd
exit /b 0

:infra_failed
echo Local infrastructure setup failed.
popd
exit /b 1
