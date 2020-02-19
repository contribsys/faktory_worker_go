test:
	go test ./...

work:
	go run test/main.go

cover:
	go test -cover -coverprofile .cover.out .
	go tool cover -html=.cover.out -o coverage.html
	open coverage.html

lint:
	go vet .

.PHONY: work test cover
