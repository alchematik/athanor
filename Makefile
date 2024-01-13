buf/generate:
	buf generate

buf/push:
	cd proto && buf push && cd -

state/show: build/provider/gcp
	go run ./cmd/athanor/main.go state show ./test/consumer/json/blueprint.json

build/translator/go:
	cd ../athanor-go && go build -o ../athanor/.translators/go/v0.0.1/translator ./cmd/translator && cd -

build/provider/gcp:
	cd ../athanor-go && go build -o ../athanor/.backends/gcp/v0.0.1/provider ./example/provider && cd -
