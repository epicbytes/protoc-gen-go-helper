#build:
#	go build -o /usr/local/bin/protoc-gen-go-helpers main.go
#	go install .

.PHONY: install gen-tag

install:
	go install .
