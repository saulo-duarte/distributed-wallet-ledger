resource "aws_sns_topic" "ledger_events" {
  name = var.sns_topic_name
}

resource "aws_sqs_queue" "wallet_balance_projections_dlq" {
  name                      = "${var.sqs_wallet_balance_queue_name}-dlq"
  message_retention_seconds = 1209600 # 14 days
}

resource "aws_sqs_queue" "wallet_balance_projections" {
  name                       = var.sqs_wallet_balance_queue_name
  visibility_timeout_seconds = 30
  message_retention_seconds  = 86400 # 1 day

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.wallet_balance_projections_dlq.arn
    maxReceiveCount     = 5
  })
}

resource "aws_sns_topic_subscription" "wallet_balance_projections" {
  topic_arn = aws_sns_topic.ledger_events.arn
  protocol  = "sqs"
  endpoint  = aws_sqs_queue.wallet_balance_projections.arn
}

resource "aws_sqs_queue_policy" "wallet_balance_projections" {
  queue_url = aws_sqs_queue.wallet_balance_projections.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowSNSToPublish"
        Effect    = "Allow"
        Principal = "*"
        Action    = "sqs:SendMessage"
        Resource  = aws_sqs_queue.wallet_balance_projections.arn
        Condition = {
          ArnEquals = {
            "aws:SourceArn" = aws_sns_topic.ledger_events.arn
          }
        }
      }
    ]
  })
}
