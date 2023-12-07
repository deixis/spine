install:
		brew install dep && dep ensure

dist: build
		dep ensure

update: build
		dep ensure -update

build:
		go build ./...

test:
		go test -v -race ./...

gen-cert:
		# openssl req -x509 -nodes -newkey rsa:2048 -keyout 127.0.0.1.key -out 127.0.0.1.crt -days 3650
		openssl req \
			-x509 -nodes -newkey rsa:2048 \
			-out net/grpc/test/localhost.crt \
			-keyout net/grpc/test/localhost.key \
			-subj "/C=CH/ST=Zurich/L=Zurich/O=Foo/OU=Bar/emailAddress=demo@example.com/CN=localhost/subjectAltName=DNS:localhost"

proto-build:
		@find . -iname '*.proto' -not -path "./vendor/*" | xargs -I '{}' protoc \
			--go_out=$(shell dirname '{}') \
			--go_opt=paths=source_relative \
			--go-grpc_out=$(shell dirname '{}') \
			--go-grpc_opt=paths=source_relative '{}'