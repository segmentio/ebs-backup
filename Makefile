V := $(shell git describe --tags --always)

export LAMBDA_S3_BUCKET := segment-lambdas
export LAMBDA_S3_KEY := ebs-backup/$(V).zip
export CIRCLE_WORKFLOW_ID ?= $(shell uuidgen | tr '[A-F]' '[a-f]')


ifndef CI
AWS_EXEC_OPS_WRAPPER := aws-okta exec ops-privileged --
AWS_EXEC_DEV_WRAPPER := aws-okta exec development-privileged --
endif

test:
	go test --cover --race ./internal/...

dist/ebs-backup-lambda: functions/ebs-backup/*.go internal/engine/*.go
	env GOOS=linux GOARCH=amd64 go build -o dist/ebs-backup-lambda ./functions/ebs-backup

dist/lambda.zip: dist/ebs-backup-lambda
	cd dist && zip -u lambda.zip ebs-backup-lambda

dist: dist/lambda.zip

push: dist/lambda.zip
	$(AWS_EXEC_OPS_WRAPPER) aws s3 cp ./dist/lambda.zip s3://$(LAMBDA_S3_BUCKET)/$(LAMBDA_S3_KEY)

test_aws:
	$(AWS_EXEC_DEV_WRAPPER) go test -v ./test/aws

clean:
	rm -fr dist

.PHONY: test test_aws clean