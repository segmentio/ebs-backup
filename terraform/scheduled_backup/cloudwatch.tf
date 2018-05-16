resource "aws_cloudwatch_event_rule" "ebs_backup" {
  # Do not use a name_prefix here or the rule will be created twice.
  # See https://github.com/terraform-providers/terraform-provider-aws/issues/4547
  name = "ebs-backup-${var.volume_name}"

  description         = "Back up ${var.volume_name} every ${var.frequency}"
  schedule_expression = "rate(${var.frequency})"
  is_enabled          = "${var.enable_event_rule}"
}

resource "aws_cloudwatch_event_target" "ebs_backup" {
  rule = "${aws_cloudwatch_event_rule.ebs_backup.name}"
  arn  = "${aws_lambda_function.ebs_backup.arn}"
}
