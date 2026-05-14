set -e
apt-get update -qq && apt-get install -y -qq protobuf-compiler
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
export PATH=$PATH:$(go env GOPATH)/bin
protoc --go_out=. --go_opt=module=github.com/google/wisdom --go-grpc_out=. --go-grpc_opt=module=github.com/google/wisdom proto/cortex.proto
go build -o cortex-server ./cmd/cortex/main.go
