provider "aws" {
  region = "ap-northeast-1"
}

terraform {
  required_version = "0.13.2"
}

variable "slack_channel" {}
variable "slack_webhook_url" {}
variable "lambda_function_image_uri" {}

variable "lambda_function_name" {
  default = "aws_cost_notification"
}

resource "aws_iam_role_policy" "cost_notification" {
  name   = "${var.lambda_function_name}_role_policy"
  role   = aws_iam_role.iam_role_for_lambda.id
  policy = file("./cost_notification_role_policy.json")
}

resource "aws_iam_role" "iam_role_for_lambda" {
  name = "${var.lambda_function_name}_role"

  assume_role_policy = file("./lambda_assume_role_policy.json")

  tags = {
    "Name" = var.lambda_function_name
  }
}

resource "aws_lambda_function" "cost_notification" {
  function_name = "${var.lambda_function_name}_lambda"
  image_uri     = var.lambda_function_image_uri
  package_type  = "Image"
  role          = aws_iam_role.iam_role_for_lambda.arn
  timeout       = 5

  tracing_config {
    mode = "Active"
  }

  environment {
    variables = {
      "SLACK_CHANNEL"     = var.slack_channel
      "SLACK_WEBHOOK_URL" = var.slack_webhook_url
    }
  }

  tags = {
    "Name" = var.lambda_function_name
  }
}

resource "aws_lambda_permission" "cost_notification" {
  statement_id  = "AllowExecutionFromEventBridge"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.cost_notification.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.cost_notification.arn
}

resource "aws_cloudwatch_event_rule" "cost_notification" {
  name                = "${var.lambda_function_name}_event_bridge_rule"
  description         = "cost notification schedule"
  schedule_expression = "cron(0 0 * * ? *)"

  tags = {
    "Name" = var.lambda_function_name
  }
}

resource "aws_cloudwatch_event_target" "cost_notification" {
  target_id = "cost_notification"
  rule      = aws_cloudwatch_event_rule.cost_notification.name
  arn       = aws_lambda_function.cost_notification.arn
}
