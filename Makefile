VPS ?= user@your-vps-here
SERVER_ADDR ?= your-server:4444
build-server:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server_linux ./cmd/server/

build-client-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o client_linux -ldflags "-X main.serverAddr=$(SERVER_ADDR)" ./cmd/client/

build-client-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o client.exe -ldflags "-X main.serverAddr=$(SERVER_ADDR) -H=windowsgui" ./cmd/client/
deploy-server: build-server
	scp server_linux $(VPS):/root/server.new
	ssh $(VPS) "mv /root/server.new /root/server"

build-client-mac:
	go build -o client_mac -ldflags "-X main.serverAddr=$(SERVER_ADDR)" ./cmd/client/

clients: build-client-linux build-client-windows build-client-mac

deploy: deploy-server clients
