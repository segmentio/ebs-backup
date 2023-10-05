# ebs-backup - a small program to snapshot EBS volumes by tag

> **Note**  
> Segment has paused maintenance on this project, but may return it to an active status in the future. Issues and pull requests from external contributors are not being considered, although internal contributions may appear from time to time. The project remains available under its open source license for anyone to use.

## Features

- Keeps up to _N_ snapshots
- Copies tags from volumes to snapshots
- Safeguards against "pending" snapshots
- Available both as a command-line program and Lambda function

## Command-line example

Back up attached volumes tagged with `Name=db-*` and attached to `/dev/xvdf`,
retaining up to 3 snapshots per volume.

```bash
$ ebs-backup --name 'db-*' --device /dev/xvdf --limit 3
```

The program will back up all volumes that match the following criteria:

- tagged with `Name = "db-*"`
- attachment state is `"attached"`
- attachment device is `"/dev/xvdf"`
- have no "pending" snapshots being created

## Testing

A full end-to-end test suite is located in `test/aws` subdirectory.  See the
`test_aws` target in the `Makefile`.

## Deployment

The Lambda function is automatically uploaded by CircleCI to S3 at each build.
The filename pattern is as follows:
`s3://${BUCKET_NAME}/ebs-backup/ebs-backup-lambda-${VERSION}.zip`

## Terraform module

A useful Terraform module for deploying ebs-backup on AWS is located in the
`terraform/scheduled_backup` subdirectory. See the `input.tf` file for supported
variables.

## Locating the S3 Lambda function

S3 bucket and key locations for the most recent release can be found in Parameter Store in the segment-ops AWS account.  These can be useful for provisioning.  See the `update_parameter_store` target in the `Makefile`.

* S3 bucket: segment/ebs_backup/lambda_s3_bucket
* S3 key: segment/ebs_backup/lambda_s3_key
