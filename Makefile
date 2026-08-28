.PHONY: build test-build test-up test-down seed workload watch \
	test-locks test-deadlock test-long-tx test-idle-tx test-errors \
	test-connections test-growth test-failure test-all clean-test

build:
	go build -o dbwatch ./cmd/dbwatch

test-build:
	go build -o dbwatch-test ./test/cmd/dbwatch-test

# Bring up the dedicated test PostgreSQL container (test/postgres/).
test-up:
	docker compose -f test/postgres/docker-compose.yml up -d
	@echo "waiting for healthy..."
	@until [ "$$(docker inspect -f '{{.State.Health.Status}}' dbwatch-test-pg 2>/dev/null)" = "healthy" ]; do sleep 1; done
	@echo "test environment is up (5 databases on localhost:5433)"

test-down:
	docker compose -f test/postgres/docker-compose.yml down

seed: test-build
	DBWATCH_TEST_ENV=true ./dbwatch-test seed

workload: test-build
	DBWATCH_TEST_ENV=true ./dbwatch-test workload -duration 5m

# Watch the test fleet with the real dbwatch TUI.
watch: build
	./dbwatch start -config test/dbwatch.test.yaml -interval 3s

test-locks: test-build
	DBWATCH_TEST_ENV=true ./dbwatch-test scenario locks

test-deadlock: test-build
	DBWATCH_TEST_ENV=true ./dbwatch-test scenario deadlock

test-long-tx: test-build
	DBWATCH_TEST_ENV=true ./dbwatch-test scenario long-transaction

test-idle-tx: test-build
	DBWATCH_TEST_ENV=true ./dbwatch-test scenario idle-transaction

test-errors: test-build
	DBWATCH_TEST_ENV=true ./dbwatch-test scenario query-errors

test-connections: test-build
	DBWATCH_TEST_ENV=true ./dbwatch-test scenario connection-pressure

test-growth: test-build
	DBWATCH_TEST_ENV=true ./dbwatch-test scenario growth

test-failure: test-build
	DBWATCH_TEST_ENV=true ./dbwatch-test scenario database-failure

test-all: test-build
	DBWATCH_TEST_ENV=true ./dbwatch-test scenario all

# Only ever touches the dedicated test container + its volumes -- never
# the demo environment at the repo root (docker-compose.yml).
clean-test:
	docker compose -f test/postgres/docker-compose.yml down -v
	rm -f dbwatch-test
