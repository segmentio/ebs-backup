
test:
	@go test --cover --race ./internal/...

dist:
	@mkdir -p dist
	@apex build ebs-backup > dist/lambda.zip

clean:
	rm -fr dist

.PHONY: test
