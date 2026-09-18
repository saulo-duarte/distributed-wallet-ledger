variable "aws_region" {
  type        = string
  description = "AWS region"
  default     = "us-east-1"
}

variable "aws_endpoint" {
  type        = string
  description = "Local emulator endpoint (ministack)"
  default     = "http://localhost:4566"
}

variable "dynamodb_table_name" {
  type        = string
  description = "Name of the DynamoDB projections table"
  default     = "goledge_projections"
}

variable "sns_topic_name" {
  type        = string
  description = "Name of the SNS ledger events topic"
  default     = "ledger-events"
}

variable "sqs_wallet_balance_queue_name" {
  type        = string
  description = "Name of the SQS queue for wallet balance projections"
  default     = "wallet-balance-projections-queue"
}
