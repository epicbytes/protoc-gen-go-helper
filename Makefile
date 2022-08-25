build:
	protoc -I./common --go_out=./common --go_opt=paths=source_relative common.proto
	#protoc-go-inject-tag -input=./*.pb.go
	ls ./*.pb.go | xargs -n1 -IX bash -c "gsed -e 's/,omitempty//' X > X.tmp && mv X{.tmp,}"
	go build -o /usr/local/bin/protoc-gen-go-helpers main.go
	go install .