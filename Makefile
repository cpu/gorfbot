RUN_LOGLEVEL = INFO
BUILD = $(GO_ENV) go build
TEST_RACE = $(GO_ENV) GORACE=halt_on_error=1 go test -race
TEST = go test -coverprofile=coverage.out
TEST_COV = go tool cover -html=coverage.out

CMD = gorfbot

all: $(CMD)

$(CMD):
	$(BUILD) ./cmd/$(@)

clean:
	rm -f ./$(CMD)

run: $(CMD)
	./$(CMD) -loglevel $(RUN_LOGLEVEL)

debug:
	dlv debug ./cmd/$(CMD)

lint:
	golangci-lint run

test:
	$(TEST) ./...

test-race:
	$(TEST_RACE) ./...

test-cov: test
	$(TEST_COV)

snapshot: $(CMD) test
	goreleaser --snapshot --skip-publish --rm-dist

.PHONY: clean run debug lint test test-race test-cov snapshot
