test:
  # Ensures that our weighted testing uses the same random seed
  # so results are predictable and assertable.
	GODEBUG=randautoseed=0 go test ./...

work:
	go run test/main.go

cover:
	go test -cover -coverprofile .cover.out .
	go tool cover -html=.cover.out -o coverage.html
	open coverage.html

lint:
	golangci-lint run

.PHONY: work test cover
