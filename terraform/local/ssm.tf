resource "aws_ssm_parameter" "database_url" {
  name        = "/goledge/DATABASE_URL"
  description = "PostgreSQL Database Connection URL"
  type        = "SecureString"
  value       = "postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable"
}

resource "aws_ssm_parameter" "api_addr" {
  name        = "/goledge/API_ADDR"
  description = "API Bind Address"
  type        = "String"
  value       = ":8080"
}

resource "aws_ssm_parameter" "migrations_path" {
  name        = "/goledge/MIGRATIONS_PATH"
  description = "Database Migrations Path"
  type        = "String"
  value       = "migrations"
}

resource "aws_ssm_parameter" "log_level" {
  name        = "/goledge/LOG_LEVEL"
  description = "Application Log Level"
  type        = "String"
  value       = "debug"
}

resource "aws_ssm_parameter" "log_format" {
  name        = "/goledge/LOG_FORMAT"
  description = "Application Log Format"
  type        = "String"
  value       = "text"
}

resource "aws_ssm_parameter" "dynamodb_endpoint" {
  name        = "/goledge/DYNAMODB_ENDPOINT"
  description = "DynamoDB Endpoint"
  type        = "String"
  value       = var.aws_endpoint
}

resource "aws_ssm_parameter" "dynamodb_region" {
  name        = "/goledge/DYNAMODB_REGION"
  description = "DynamoDB Region"
  type        = "String"
  value       = var.aws_region
}

resource "aws_ssm_parameter" "dynamodb_table" {
  name        = "/goledge/DYNAMODB_TABLE"
  description = "DynamoDB Projections Table Name"
  type        = "String"
  value       = aws_dynamodb_table.projections.name
}

resource "aws_ssm_parameter" "aws_endpoint" {
  name        = "/goledge/AWS_ENDPOINT"
  description = "AWS Services Local Endpoint"
  type        = "String"
  value       = var.aws_endpoint
}

resource "aws_ssm_parameter" "aws_region" {
  name        = "/goledge/AWS_REGION"
  description = "AWS Region"
  type        = "String"
  value       = var.aws_region
}

resource "aws_ssm_parameter" "sns_topic_arn" {
  name        = "/goledge/SNS_TOPIC_ARN"
  description = "SNS Topic ARN for Ledger Events"
  type        = "String"
  value       = aws_sns_topic.ledger_events.arn
}

resource "aws_ssm_parameter" "sqs_wallet_balance_queue_url" {
  name        = "/goledge/SQS_WALLET_BALANCE_QUEUE_URL"
  description = "SQS Queue URL for Wallet Balance Projections"
  type        = "String"
  value       = aws_sqs_queue.wallet_balance_projections.url
}
