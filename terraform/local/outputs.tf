output "dynamodb_table_name" {
  value       = aws_dynamodb_table.projections.name
  description = "DynamoDB projections table name"
}

output "sns_topic_arn" {
  value       = aws_sns_topic.ledger_events.arn
  description = "SNS ledger events topic ARN"
}

output "sqs_wallet_balance_queue_url" {
  value       = aws_sqs_queue.wallet_balance_projections.id
  description = "SQS wallet balance projections queue URL"
}

output "sqs_wallet_balance_queue_arn" {
  value       = aws_sqs_queue.wallet_balance_projections.arn
  description = "SQS wallet balance projections queue ARN"
}

output "sqs_wallet_balance_dlq_url" {
  value       = aws_sqs_queue.wallet_balance_projections_dlq.id
  description = "SQS wallet balance projections DLQ URL"
}
