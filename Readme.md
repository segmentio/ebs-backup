# ebs-backup - a small program to backup ebs volumes by tag names.

## Features

- Keep up to N snapshots
- Copies volume tags
- Safeguards against "pending" snapshots
- Flexible deployment (Lambda/ECS/EC2 etc..)

## Example

Backup volumes tagged with `Name=db-*` and keep up to 3 snapshots for each volume.

```bash
$ ebs-backup --name 'db-*' --device /dev/xvdf --limit 3
```

The program will lookup all volumes that match the following:

- tagged with `Name = "db-*"`
- attachment state is `"attached"`
- attachment device is `"/dev/xvdf"`
- have no "pending" snapshots being created

## Deployment

This service is deployed by Terraform.

Create a tag for your new release :

```bash
git tag v0.1.1
```

To build a new version :

```bash
make dist
```

Push to s3 :

```bash
make push
```

Then update Terraform : https://github.com/segment-infra/services/blob/master/dedupe/main.tf#L186
