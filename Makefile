.PHONY: build deploy
build:
	GOOS=linux GOARCH=amd64 go build -o deploy/app app.go

deploy: build
	ansible-playbook -i ansible/hosts ansible/deploy.yml -e 'ansible_python_interpreter=/usr/bin/python3'

