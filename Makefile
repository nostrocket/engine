run:
	go mod tidy
	go run cmd/engine/*.go

reset:
	rm -rf ~/nostrocket/data
	go mod tidy
	go run cmd/engine/*.go