vendor:
	dep ensure -vendor-only -v

build: vendor
	go build -o envoy-client cmd/envoy_client.go
	go build -o envoy-server cmd/envoy_server.go
