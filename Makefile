.PHONY: build
build:
	GOOS=linux GOARCH=amd64 go build -o deploy/app app.go

zip:
	tar -zcvf deploy.tar.gz deploy

