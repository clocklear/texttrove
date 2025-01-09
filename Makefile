./bin/gosec:
	wget -O - -q https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s latest

.PHONY: security
security: ./bin/gosec
	./bin/gosec ./...

.PHONY: test
test:
	go test -race ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: vendor
vendor:
	go mod tidy && go mod vendor