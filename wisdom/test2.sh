set -e
apt-get update -qq && apt-get install -y -qq protobuf-compiler
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
export PATH=$PATH:$(go env GOPATH)/bin
protoc --go_out=. --go_opt=module=github.com/google/wisdom \
       --go-grpc_out=. --go-grpc_opt=module=github.com/google/wisdom \
       proto/cortex.proto proto/thalamus.proto proto/mastery.proto \
       proto/researcher.proto proto/curriculum.proto proto/integrations.proto \
       proto/entity.proto proto/pubsub_events.proto
find pkg -name 'stubs.go' -delete || true
go build -o researcher-job ./cmd/researcher/main.go
