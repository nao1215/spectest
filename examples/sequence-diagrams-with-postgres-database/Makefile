.PHONY: postgres-dev test

postgres-dev:
	docker stop spectest_postgres || true && docker rm spectest_postgres || true
	docker run -d -e POSTGRES_PASSWORD=postgres -e POSTGRES_USER=postgres -e POSTGRES_DB=spectest -p 5432:5432 --name spectest_postgres postgres:9.6

test:
	POSTGRES_DSN="host=localhost port=5432 user=postgres password=postgres dbname=spectest sslmode=disable" go test ./... -v
