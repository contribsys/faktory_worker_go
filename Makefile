work:
	go run test/main.go

# use TLS
swork:
	FAKTORY_URL=tcp://localhost.contribsys.com:7419 FAKTORY_PROVIDER=FAKTORY_URL go run test/main.go
