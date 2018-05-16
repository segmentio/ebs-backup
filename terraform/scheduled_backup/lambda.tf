resource "aws_lambda_function" "ebs_backup" {
  function_name = "ebs-backup"
  handler       = "ebs-backup-lambda"
  role          = "${aws_iam_role.ebs_backup.arn}"
  s3_bucket     = "${var.lambda_s3_bucket}"
  s3_key        = "${var.lambda_s3_key}"
  runtime       = "go1.x"

  environment {
    variables {
      COPY_TAGS      = "${var.copy_tags}"
      SNAPSHOT_LIMIT = "${var.snapshot_limit}"
      VOLUME_DEVICES = "${join(" ", var.device_names)}"
      VOLUME_NAME    = "${var.volume_name}"
    }
  }
}

resource "aws_lambda_permission" "ebs_backup" {
  statement_id  = "AllowExecutionFromCloudWatch"
  action        = "lambda:InvokeFunction"
  function_name = "${aws_lambda_function.ebs_backup.function_name}"
  principal     = "events.amazonaws.com"
  source_arn    = "${aws_cloudwatch_event_rule.ebs_backup.arn}"
}

resource "aws_iam_role" "ebs_backup" {
  name_prefix = "ebs_backup"

  assume_role_policy = <<POLICY
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": "sts:AssumeRole",
            "Principal": {
                "Service": "lambda.amazonaws.com"
            },
            "Effect": "Allow"
        }
    ]
}
POLICY
}

resource "aws_iam_role_policy" "ebs_backup" {
  name = "ebs_backup"
  role = "${aws_iam_role.ebs_backup.name}"

  policy = <<POLICY
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ec2:DescribeVolumes",
                "ec2:DescribeSnapshots",
                "ec2:CreateSnapshot",
                "ec2:CreateTags",
                "ec2:DeleteSnapshot"
            ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "logs:CreateLogGroup",
                "logs:CreateLogStream",
                "logs:PutLogEvents"
            ],
            "Resource": "*"
        }
    ]
}
POLICY
}
