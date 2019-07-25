## Deploy 

OS: Ubuntu 18.04

- Fix **ansible/hosts.**
- Run ansible-playbook
```sh
$ ansible-playbook --private-key=XXXX -i ansible/hosts ansible/deploy.yml \
    -e 'ansible_python_interpreter=/usr/bin/python3'
```
