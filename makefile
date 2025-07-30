include .env
export 

dev:
	@echo "Starting test coverage and watch mode..."
	@echo "Press Ctrl+C to stop both processes"
	@trap 'echo "Stopping processes..."; kill %1 %2 2>/dev/null; exit' INT; \
	$(MAKE) test/coverage & \
	$(MAKE) test/watch & \
	wait
	
test:
	richgo test -v ./...

test/watch:
	find . -name '*.go' | entr -cr richgo test -v ./...

test/coverage:
	mkdir -p tmp
	go test -coverprofile=tmp/coverage.out && go tool cover -html=tmp/coverage.out -o tmp/coverage.html
	@echo "Coverage report generated: tmp/coverage.html"
