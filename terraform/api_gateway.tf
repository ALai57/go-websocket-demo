
locals {
  lambda_name = "go-websocket"

  db_name = "go_websocket"
  db_host = "andrewslai-postgres.cwvfukjbn65j.us-east-1.rds.amazonaws.com"
  db_port = "5432"
  db_user = "go_websocket_user"

  log_level        = "INFO"
  private_subnet_1 = "subnet-015cf9c5fe2cca75e"
}

#####################################################
# Networking
#####################################################
data "aws_vpc" "default" {
  default = true
}

// Public subnets
data "aws_subnets" "all" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

data "aws_security_group" "default" {
  vpc_id = data.aws_vpc.default.id
  name   = "default"
}

data "aws_security_group" "db_allow" {
  vpc_id = data.aws_vpc.default.id
  name   = "allow-default-vpc"
}

// Manually set this up because a Lambda on a VPC
// must be associated with a private subnet
data "aws_subnet" "privnet" {
  id = local.private_subnet_1
}

// Also set up a VPC endpoint for Secretsmanager manually - this
// gave the Lambda and private traffic on the private subnet
// access to the secretsmanager service without going over
// the public internet.

#####################################################
# Lambda
#####################################################
resource "aws_lambda_function" "go_websocket_lambda" {
  function_name = local.lambda_name
  role          = aws_iam_role.iam_for_lambda.arn
  handler       = "bootstrap"
  filename      = "../go-websocket.zip"
  runtime       = "provided.al2023"
  architectures = ["arm64"]

  environment {
    variables = {
      DB_NAME   = local.db_name
      DB_HOST   = local.db_host
      DB_PORT   = local.db_port
      DB_USER   = local.db_user
      LOG_LEVEL = local.log_level
      #DB_PASSWORD = local.secrets
      DB_PASSWORD_ARN = aws_secretsmanager_secret.secrets.arn
    }
  }

  timeout = 15
  vpc_config {
    subnet_ids         = [data.aws_subnet.privnet.id]
    security_group_ids = ["${data.aws_security_group.default.id}", "${data.aws_security_group.db_allow.id}"]
  }
}

resource "aws_iam_role" "iam_for_lambda" {
  name               = "${local.lambda_name}-iam-role"
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "Lambda",
      "Effect": "Allow",
      "Principal": { "Service": "lambda.amazonaws.com" },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_policy" "lambda_permissions" {
  name        = "go_websocket_lambda_permissions"
  path        = "/"
  description = "IAM policy for go_websocket_lambda"
  policy      = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "SecretsManager",
      "Action": [
        "secretsmanager:GetSecretValue"
      ],
      "Effect": "Allow",
      "Resource": [
        "${aws_secretsmanager_secret.secrets.arn}"
      ]
    }
  ]
}
EOF
}

resource "aws_iam_role_policy_attachment" "lambda_permissions" {
  role       = aws_iam_role.iam_for_lambda.name
  policy_arn = aws_iam_policy.lambda_permissions.arn
}

# https://stackoverflow.com/a/64044160
# AWSLambdaVPCAccessExecutionRole has redundant logging permissions
resource "aws_iam_role_policy_attachment" "AWSLambdaVPCAccessExecutionRole" {
  role       = aws_iam_role.iam_for_lambda.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"
}

#####################################################
# Logging
#####################################################
resource "aws_cloudwatch_log_group" "example" {
  name              = "/aws/lambda/${local.lambda_name}"
  retention_in_days = 14
}

#####################################################
# API GW
#####################################################

# https://awstip.com/websocket-api-gateway-with-terraform-8a509585121d
resource "aws_apigatewayv2_api" "websocket_api" {
  name                       = "${local.lambda_name}-gateway"
  protocol_type              = "WEBSOCKET"
  route_selection_expression = "$request.body.action"


  # To add Authorization, look at these resources. For now, this does not have authorization
  # https://docs.aws.amazon.com/apigateway/latest/developerguide/api-gateway-lambda-authorizer-output.html
  # https://docs.aws.amazon.com/apigateway/latest/developerguide/api-gateway-api-key-source.html
}
# https://github.com/misikdmytro/go-ws-api-gateway/blob/master/internal/handler/connect.go

# NOTE: there is some kind of broken lifecycle issues here - I actually needed
# to run twice, presumably, because one element depended on one that wasn't yet
# created

resource "aws_apigatewayv2_integration" "lambda" {
  api_id           = aws_apigatewayv2_api.websocket_api.id
  integration_type = "AWS_PROXY"


  connection_type           = "INTERNET"
  content_handling_strategy = "CONVERT_TO_TEXT"
  description               = "Lambda integration"
  integration_method        = "POST"
  integration_uri           = aws_lambda_function.go_websocket_lambda.invoke_arn
  passthrough_behavior      = "WHEN_NO_MATCH"

}

resource "aws_apigatewayv2_route" "routes" {
  for_each  = toset(["$default", "$connect", "$disconnect"])
  api_id    = aws_apigatewayv2_api.websocket_api.id
  route_key = each.key
  target    = "integrations/${aws_apigatewayv2_integration.lambda.id}"
}

resource "aws_apigatewayv2_deployment" "go_websocket_gateway" {
  depends_on = [
    aws_apigatewayv2_integration.lambda,
  ]

  api_id = aws_apigatewayv2_api.websocket_api.id
}

resource "aws_apigatewayv2_stage" "prod" {
  api_id = aws_apigatewayv2_api.websocket_api.id
  name   = "prod"
}

#######################################
# Permissions
#######################################
resource "aws_lambda_permission" "apigw" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.go_websocket_lambda.function_name
  principal     = "apigateway.amazonaws.com"

  # The /*/* portion grants access from any method on any resource
  # within the API Gateway "REST API".
  source_arn = "${aws_apigatewayv2_api.websocket_api.execution_arn}/*/*"
}


output "base_url" {
  value = aws_apigatewayv2_deployment.go_websocket_gateway.id
}

#######################################
# Database access
#######################################
resource "aws_secretsmanager_secret" "secrets" {
  name = "go_websocket_secrets"
}
