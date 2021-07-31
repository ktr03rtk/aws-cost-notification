resource "aws_iam_role_policy" "cost_notification" {
  name   = "${var.lambda_function_name}_role_policy"
  role   = aws_iam_role.iam_role_for_lambda.id
  policy = file("${path.module}/cost_notification_role_policy.json")
}

resource "aws_iam_role" "iam_role_for_lambda" {
  name = "${var.lambda_function_name}_role"

  assume_role_policy = file("${path.module}/lambda_assume_role_policy.json")

  tags = {
    "Name" = var.lambda_function_name
  }
}
