.PHONY: mysql-dev test

mysql-dev:
	docker stop spectest_mysql || true && docker rm spectest_mysql || true
	docker run -d \
	    -e MYSQL_ROOT_PASSWORD=mysql \
	    -e MYSQL_PASSWORD=mysql \
	    -e MYSQL_USER=mysql \
	    -e MYSQL_DATABASE=spectest \
	    -p 3306:3306 \
	    --name apitest_mysql \
	    mysql:8.0

test:
	MYSQL_DSN="mysql:mysql@/spectest" go test ./... -v
