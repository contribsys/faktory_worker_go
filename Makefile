test:
	go test -race ./...

work:
	go run -race test/main.go

.PHONY: work test
