variable "lambda_s3_bucket" {
  type        = "string"
  description = "S3 bucket containing EBS backup Lambda function"
}

variable "lambda_s3_key" {
  type        = "string"
  description = "S3 key pointing to EBS backup Lambda function"
}

variable "copy_tags" {
  default     = true
  description = "Copy tags from EBS volume to snapshot"
}

variable "frequency" {
  type        = "string"
  description = "Frequency at which backup is run (see https://docs.aws.amazon.com/AmazonCloudWatch/latest/events/ScheduledEvents.html#RateExpressions for legal values)"
  default     = "2 hours"
}

variable "snapshot_limit" {
  default     = 2
  description = "Number of most recent snapshots to retain"
}

variable "volume_name" {
  type        = "string"
  description = "Value of `Name` tag on EBS volumes to match"
}

variable "device_names" {
  type        = "list"
  description = "List of device attachment names to match (e.g. `/dev/xvdf`)"
}

variable "enable_event_rule" {
  default     = true
  description = "Enable event rule (not normally disabled)"
}
