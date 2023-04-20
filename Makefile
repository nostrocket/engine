run:
	go mod tidy
	go run cmd/engine/*.go

reset:
	rm -rf ~/nostrocket/data
	go mod tidy
	go run cmd/engine/*.go

debug:
	rm -rf ~/nostrocket/data
	go mod tidy
	NOSTROCKET_DEBUG=true ~/go/bin/dlv debug cmd/engine/*.go