provider "aws" {
  assume_role {
    role_arn = "${var.assume_role_arn}"
  }
}

variable "name" {}

variable "lambda_s3_bucket" {}
variable "lambda_s3_key" {}

variable "build_username" {}

variable "workflow_id" {}

variable "assume_role_arn" {
  default = ""
}

variable "build_url" {
  default = "N/A"
}

variable "pull_request_url" {
  default = "N/A"
}

data "aws_region" "current" {}

data "aws_availability_zones" "available" {}

data "aws_ami" "linux" {
  most_recent = true

  filter {
    name   = "name"
    values = ["amzn-ami-hvm*"]
  }

  filter {
    name   = "owner-alias"
    values = ["amazon"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  filter {
    name   = "architecture"
    values = ["x86_64"]
  }

  filter {
    name   = "root-device-type"
    values = ["ebs"]
  }
}

locals {
  availability_zone = "${data.aws_availability_zones.available.names[0]}"
  device_name       = "/dev/xvdf"
}

resource "aws_vpc" "test" {
  cidr_block = "172.16.0.0/16"

  tags {
    Name    = "${var.name}"
    Creator = "${var.build_username}"
  }
}

resource "aws_subnet" "test" {
  vpc_id            = "${aws_vpc.test.id}"
  cidr_block        = "172.16.42.0/24"
  availability_zone = "${local.availability_zone}"

  tags {
    Name    = "${var.name}"
    Creator = "${var.build_username}"
  }
}

resource "aws_instance" "test" {
  ami           = "${data.aws_ami.linux.id}"
  subnet_id     = "${aws_subnet.test.id}"
  instance_type = "t2.micro"
}

resource "aws_ebs_volume" "data" {
  availability_zone = "${local.availability_zone}"
  type              = "gp2"
  size              = "100"

  tags {
    Name    = "${var.name}"
    Creator = "${var.build_username}"
  }
}

resource "aws_volume_attachment" "data" {
  volume_id   = "${aws_ebs_volume.data.id}"
  instance_id = "${aws_instance.test.id}"
  device_name = "${local.device_name}"
}

module "scheduled_backup" {
  source = "../../terraform/scheduled_backup"

  lambda_s3_bucket  = "${var.lambda_s3_bucket}"
  lambda_s3_key     = "${var.lambda_s3_key}"
  volume_name       = "${var.name}"
  device_names      = ["${local.device_name}"]
  enable_event_rule = false
}

output "backup_function_name" {
  value = "${module.scheduled_backup.backup_function_name}"
}
