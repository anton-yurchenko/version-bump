I := "⚪"
E := "🔴"

.PHONY: lint
lint: $(GO_LINTER)
	@echo "$(I) installing dependencies..."
	@go get ./... || (echo "$(E) 'go get' error"; exit 1)
	@echo "$(I) updating imports..."
	@go mod tidy || (echo "$(E) 'go mod tidy' error"; exit 1)
	@echo "$(I) vendoring..."
	@go mod vendor || (echo "$(E) 'go mod vendor' error"; exit 1)
	@echo "$(I) linting..."
	@golangci-lint run ./... || (echo "$(E) linter error"; exit 1)
	$(MAKE) test

.PHONY: init
init:
	@echo "$(I) initializing..."
	@mv .vscode/launch-template.json .vscode/launch.json 2>/dev/null || :
	@rm -rf go.mod go.sum ./vendor ./mocks
	@go mod init $$(pwd | awk -F'/' '{print $$NF}')

.PHONY: codecov
codecov: test
	@go tool cover -html=coverage.txt || (echo "$(E) 'go tool cover' error"; exit 1)

.PHONY: mock
mock:
	@echo "$(I) regenerating mocks package..."
	@docker run -v "$(PWD)":/src -w /src vektra/mockery --name=Repository --dir=/src/bump/
	@docker run -v "$(PWD)":/src -w /src vektra/mockery --name=Worktree --dir=/src/bump/
	@sudo chown -R $(USER):$(id -gn) mocks

.PHONY: test
test:
	@echo "$(I) unit testing..."
	@go test -v $$(go list ./... | grep -v vendor | grep -v mocks) -race -coverprofile=coverage.txt -covermode=atomic

GO_LINTER := $(GOPATH)/bin/golangci-lint
$(GO_LINTER):
	@echo "installing linter..."
	@go get -u github.com/golangci/golangci-lint/cmd/golangci-lint
