env:
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

generate-proto:
	export PATH=$$PATH:$$HOME/go/bin && protoc \
		--proto_path=proto \
		--go_out=./internal/proto \
		--go_opt=paths=source_relative \
		--go-grpc_out=./internal/proto \
		--go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=./internal/proto \
		--grpc-gateway_opt=paths=source_relative \
		--grpc-gateway_opt=generate_unbound_methods=true \
		--openapiv2_out=./api/swagger \
		--openapiv2_opt=allow_merge=true \
		--openapiv2_opt=merge_file_name=api \
		authorization.proto \
		badge.proto \
		comment.proto \
		community.proto \
		entity.proto \
		feed.proto \
		media.proto \
		moderation.proto \
		notification.proto \
		permission.proto \
		platform.proto \
		post.proto \
		report.proto \
		role.proto \
		search.proto \
		user.proto

binary-build:
	go build -o backend ./cmd/backend
	strip backend

image-build: version ?= latest
image-build:
	docker buildx build \
		--platform linux/amd64 \
		--file Dockerfile.microservice \
		--tag stormic/backend:${version} \
		.

image-push: version ?= latest
image-push:
	docker push stormic/backend:${version}

chart-build: version ?= 0.1.0
chart-build:
	helm package chart --version ${version}

chart-install: version ?= 0.1.0
chart-install:
	helm -n community install --create-namespace community community-${version}.tgz

infrastructure-up:
	docker-compose up -d

infrastructure-down:
	docker-compose down

database-create-migration: name ?= initial
database-create-migration:
	migrate create -ext sql -dir migration -seq ${name}

database-apply-migrations:
	migrate -database 'postgres://postgres:postgres@127.0.0.1:5432?sslmode=disable' -path migration up

database-delete-migrations:
	migrate -database 'postgres://postgres:postgres@127.0.0.1:5432?sslmode=disable' -path migration down