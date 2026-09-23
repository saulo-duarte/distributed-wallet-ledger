resource "aws_dynamodb_table" "projections" {
  name         = var.dynamodb_table_name
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "pk"
  range_key    = "sk"

  attribute {
    name = "pk"
    type = "S"
  }

  attribute {
    name = "sk"
    type = "S"
  }

  attribute {
    name = "created_at"
    type = "S"
  }

  local_secondary_index {
    name            = "lsi_created_at"
    range_key       = "created_at"
    projection_type = "ALL"
  }
}
