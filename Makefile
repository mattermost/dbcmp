.PHONY: test run-databases stop-databases

test: run-databases
	go test -v ./...

run-databases:
	@echo Starting docker containers
	docker-compose -f internal/testing/docker-compose.yml --project-name dbcmp up --no-recreate -d

stop-databases:
	@echo Stopping docker containers
	docker-compose -f internal/testing/docker-compose.yml --project-name dbcmp down
