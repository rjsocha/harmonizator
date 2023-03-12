all: release

release:
	CGO_ENABLED=0 go build -ldflags="-s -w" harmonizator.go
	upx --lzma harmonizator

debug:
	CGO_ENABLED=1 go build -race harmonizator.go

install-client:
	sudo install hrm /usr/local/bin/hrm
