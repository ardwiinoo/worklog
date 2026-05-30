APP_NAME=worklog
VERSION=v0.5.0
OUT=dist/$(APP_NAME)-windows-amd64-$(VERSION).exe

build-windows:
	mkdir -p dist
	go build -trimpath -ldflags="-s -w -H windowsgui" -o $(OUT) .