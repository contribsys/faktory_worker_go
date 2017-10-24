work:
	go run test/main.go

# use TLS
swork:
	FAKTORY_URL=tcp://127.0.0.1:7419 FAKTORY_PROVIDER=FAKTORY_URL go run test/main.go
