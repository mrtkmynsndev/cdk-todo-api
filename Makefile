.PHONY: .build

build:
	cd lambda && GOOS=linux GOARCH=amd64 go build -o lambdaHandler . & cd ..