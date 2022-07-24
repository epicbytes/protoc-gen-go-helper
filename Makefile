build:
	go build -o /usr/local/bin/protoc-gen-go-helpers main.go
	go install .