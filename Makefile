APP_NAME = easy-http-proxy-pool
BUILD_DIR = $(PWD)/build

build: clean
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(APP_NAME)_linux_amd64 main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(APP_NAME)_linux_arm64 main.go

mkdir:
	mkdir -p build

clean:
	rm -rf ./build
