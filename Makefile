gen:
	protoc --proto_path=proto --go_out=pb --go_opt=paths=source_relative \
    --go-grpc_out=pb --go-grpc_opt=paths=source_relative \
    proto/*.proto
clean:
	rm pb/*.go
server:
	go run cmd/server/main.go -port 8080
client:
	go run cmd/client/main.go -address 0.0.0.0:8080
test:
	go test -cover -race ./...
cert:
	cd cert; ./gen.sh; cd ..
docker_build:
	docker build -t test:latest .
docker_run:
	docker run --name test -p 8080:8080 test:latest

.PHONY: clean gen server client test cert 