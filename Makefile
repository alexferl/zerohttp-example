.PHONY: dev audit cover cover-html fmt lint generate pre-commit run test tidy update-deps docker-build docker-run

.DEFAULT: help
help:
	@echo "make dev"
	@echo "	setup development environment"
	@echo "make audit"
	@echo "	conduct quality checks"
	@echo "make cover"
	@echo "	generate coverage report"
	@echo "make cover-html"
	@echo "	generate coverage HTML report"
	@echo "make fmt"
	@echo "	fix code format issues"
	@echo "make lint"
	@echo "	run lint checks"
	@echo "make generate"
	@echo "	run mockery"
	@echo "make pre-commit"
	@echo "	run pre-commit hooks"
	@echo "make run"
	@echo "	run application"
	@echo "make test"
	@echo "	run all tests"
	@echo "make tidy"
	@echo "	clean and tidy dependencies"
	@echo "make update-deps"
	@echo "	update dependencies"
	@echo "make docker-build"
	@echo "	build docker image"
	@echo "make docker-run"
	@echo "	run docker image"

GOTESTSUM := go run gotest.tools/gotestsum@latest -f testname -- ./... -race -count=1
TESTFLAGS := -shuffle=on
COVERFLAGS := -covermode=atomic
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4

check-pre-commit:
ifeq (, $(shell which pre-commit))
	$(error "pre-commit not in $(PATH), pre-commit (https://pre-commit.com) is required")
endif

dev: check-pre-commit
	pre-commit install

audit:
	go mod verify
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

cover:
	$(GOTESTSUM) $(TESTFLAGS) $(COVERFLAGS)

cover-html:
	$(GOTESTSUM) $(TESTFLAGS) $(COVERFLAGS) -coverprofile=coverage.out
	go tool cover -html=coverage.out

fmt:
	$(GOLANGCI_LINT) fmt

lint:
	$(GOLANGCI_LINT) run

generate:
	mockery

pre-commit: check-pre-commit
	pre-commit run --all-files

run:
	go run ./cmd/server

test:
	$(GOTESTSUM) $(TESTFLAGS)

tidy:
	go mod tidy -v

update-deps: tidy
	go get -u ./...

docker-build:
	docker build -t vinylstore .

docker-run:
	docker run --name vinylstore -p 8080:8080 --rm \
		-e VINYL_BIND_ADDR=0.0.0.0:8080 \
		-e VINYL_MONGO_URI=mongodb://host.docker.internal:27017 \
		-e VINYL_REDIS_ADDR=host.docker.internal:6379 \
		vinylstore
