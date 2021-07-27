BINARY := $(notdir $(CURDIR))
GO_BIN_DIR := $(GOPATH)/bin
OSES := linux darwin windows
ARCHS := amd64

test: lint
	@go test $$(go list ./... | grep -v vendor | grep -v mocks) -race -coverprofile=coverage.txt -covermode=atomic

.PHONY: lint
lint: $(GO_LINTER)
	@go mod tidy
	@go mod vendor
	@golangci-lint run ./...

.PHONY: init
init:
	@mv .vscode/launch-template.json .vscode/launch.json 2>/dev/null || :
	@rm -rf go.mod go.sum ./vendor
	@go mod init $$(pwd | awk -F'/' '{print $$NF}')

GO_LINTER := $(GO_BIN_DIR)/golangci-lint
$(GO_LINTER):
	@echo "installing linter..."
	go get -u github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY: release
release: test
	@rm -rf ./dist
	@mkdir -p dist
	@for ARCH in $(ARCHS); do \
		for OS in $(OSES); do \
			if test "$$OS" = "windows"; then \
				GOOS=$$OS GOARCH=$$ARCH go build -o dist/$(BINARY)-$$OS-$$ARCH.exe; \
			else \
				GOOS=$$OS GOARCH=$$ARCH go build -o dist/$(BINARY)-$$OS-$$ARCH; \
			fi; \
		done; \
	done

.PHONY: codecov
codecov: test
	@go tool cover -html=coverage.txt
