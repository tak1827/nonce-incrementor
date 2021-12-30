.PHONY: test
test:
	go test -race ./...

fmt:
	go fmt ./...

lint:
	go vet ./...
