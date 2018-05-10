V=`git describe --tags --always`

export LAMBDA_S3_BUCKET := segment-lambdas
export LAMBDA_S3_KEY := ebs-backup/$(V).zip

ifdef CI
AWS_WRAPPER :=
else
AWS_WRAPPER := aws-okta exec ops-privileged --
endif

test:
	go test --cover --race ./internal/...

test-aws:
	go test -v ./test/aws

dist/ebs-backup-lambda: functions/ebs-backup/*.go internal/engine/*.go
	env GOOS=linux GOARCH=amd64 go build -o dist/ebs-backup-lambda ./functions/ebs-backup

dist/lambda.zip: dist/ebs-backup-lambda
	cd dist && zip -u lambda.zip ebs-backup-lambda

dist: dist/lambda.zip

push: dist/lambda.zip
	$(AWS_WRAPPER) aws s3 cp ./dist/lambda.zip s3://$(LAMBDA_S3_BUCKET)/$(LAMBDA_S3_KEY)

clean:
	rm -fr dist

.PHONY: test test-aws clean
