test:
	go test ./...

work:
	go run test/main.go

.PHONY: work test
