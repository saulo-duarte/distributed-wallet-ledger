resource "aws_sns_topic" "ledger_events" {
  name = var.sns_topic_name
}

resource "aws_sqs_queue" "wallet_balance_projections_dlq" {
  name                      = "${var.sqs_wallet_balance_queue_name}-dlq"
  message_retention_seconds = 1209600 # 14 days

  lifecycle {
    # MinStack may expose provider defaults for queues created outside Terraform
    # and can hang while reconciling retention on an existing local queue.
    ignore_changes = [message_retention_seconds]
  }
}

resource "aws_sqs_queue" "wallet_balance_projections" {
  name                       = var.sqs_wallet_balance_queue_name
  visibility_timeout_seconds = 30
  message_retention_seconds  = 86400 # 1 day

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.wallet_balance_projections_dlq.arn
    maxReceiveCount     = 5
  })

  lifecycle {
    # Keep local bootstrap idempotent when the queue already exists in MinStack.
    ignore_changes = [message_retention_seconds, redrive_policy]
  }
}

resource "aws_sns_topic_subscription" "wallet_balance_projections" {
  topic_arn = aws_sns_topic.ledger_events.arn
  protocol  = "sqs"
  endpoint  = aws_sqs_queue.wallet_balance_projections.arn
}
