generate: generate/aws generate/gcp

generate/aws:
	go run ./cmd/athanor/main.go provider generate -mod github.com/alchematik/athanor -out ./gen ./aws.hcl

generate/gcp:
	go run ./cmd/athanor/main.go provider generate -mod github.com/alchematik/athanor -out ./gen ./gcp.hcl

build: build/aws build/gcp

build/aws:
	go build -buildmode=plugin -o .providers/aws/v0.0.1/provider.so ./test/aws/

build/gcp:
	go build -o .providers/gcp/v0.0.1/provider ./test/gcp/

blueprint/show:
	go run ./cmd/athanor/main.go blueprint show -providers ./.providers ./gcp

