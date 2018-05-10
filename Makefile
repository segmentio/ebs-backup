V=`git describe --tags --always`

ifdef CI
AWS_WRAPPER :=
else
AWS_WRAPPER := aws-okta exec ops-privileged --
endif

test:
	go test --cover --race ./internal/...

dist/ebs-backup-lambda: functions/ebs-backup/*.go internal/engine/*.go
	env GOOS=linux GOARCH=amd64 go build -o dist/ebs-backup-lambda ./functions/ebs-backup

dist/lambda.zip: dist/ebs-backup-lambda
	cd dist && zip -u lambda.zip ebs-backup-lambda

dist: dist/lambda.zip

push: dist/lambda.zip
	$(AWS_WRAPPER) aws s3 cp ./dist/lambda.zip s3://segment-lambdas/ebs-backup/$(V).zip

clean:
	rm -fr dist

.PHONY: test clean
