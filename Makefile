app-name=starter

dep:
	go mod tidy
	go mod vendor

dev:
	go build -o bin/${app-name} cmd/main.go
	./bin/${app-name}

test:
	go test ./... -coverprofile cp.out

test-html:
	go test $(go list ./... | grep -v /mock/) -coverprofile cp.out
	go tool cover -html=cp.out
seed:
	go run ./seeder/main.go

sync-permission:
	cd ./tools/permgen&& go run main.go

build:
	set GOOS=linux&& set GOARCH=amd64&& go build -o bin/${app-name} cmd/main.go

swag:
	swag init -d app