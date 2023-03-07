all: release

release:
	go build -ldflags="-s -w" harmonizator.go
	upx -9 harmonizator

debug:
	go build -race harmonizator.go
