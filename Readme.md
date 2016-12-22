
  ebs-backup - a small program to backup ebs volumes by tag names.

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
