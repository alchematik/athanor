generate:
	go run ./cmd/athanor/main.go provider generate manifest ./test/provider/config.json

state/show:
	go run ./cmd/athanor/main.go state show ./test/consumer/json/blueprint.json

provider/generate:
	go run ./cmd/athanor/main.go provider generate ./test/provider/go/config.json

#generate/aws:
#	go run ./cmd/athanor/main.go provider generate -mod github.com/alchematik/athanor -out ./gen ./aws.hcl
#
#generate/gcp:
#	go run ./cmd/athanor/main.go provider generate -mod github.com/alchematik/athanor -out ./gen ./gcp.hcl

build: build/json #build/backend/athanor

build/aws:
	go build -o .providers/aws/v0.0.1/provider ./test/aws/

build/gcp:
	go build -o .providers/gcp/v0.0.1/provider ./test/gcp/

build/json:
	go build -o .translators/json/v0.0.1/translator ./test/json

build/backend/athanor:
	go build -o .backends/athanor/v0.0.1/backend ./cmd/backend

build/backend/gcp:
	go build -o .backends/gcp/v0.0.1/provider ./cmd/backend

build/translator/go:
	go build -o .translators/go/v0.0.1/translator ./translator/go

blueprint/show:
	go run ./cmd/athanor/main.go blueprint show -providers ./.providers ./gcp

