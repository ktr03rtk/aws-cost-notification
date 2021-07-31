provider "aws" {
  region = "ap-northeast-1"
}

terraform {
  required_version = "0.13.2"
}

module "lambda" {
  source                    = "./modules/lambda"
  slack_channel             = var.slack_channel
  slack_webhook_url         = var.slack_webhook_url
  lambda_function_image_uri = var.lambda_function_image_uri
  lambda_function_name      = var.lambda_function_name
}
