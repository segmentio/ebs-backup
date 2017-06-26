V=`git describe --tags --always`

test:
	@go test --cover --race ./internal/...

dist:
	@mkdir -p dist
	@apex build ebs-backup > dist/lambda.zip

push:
	@aws-vault exec ops -- aws s3 cp ./dist/lambda.zip s3://segment-lambdas/ebs-backup/$(V).zip

clean:
	rm -fr dist

.PHONY: test
