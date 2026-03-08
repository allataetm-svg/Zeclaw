.PHONY: run build test lint clean

run:
	go run ./cmd/zeclaw/

build:
	CGO_ENABLED=0 go build -o zeclaw ./cmd/zeclaw/

build-android-arm64:
	GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -o ../frontend/assets/backend/zeclaw-backend-arm64 ./cmd/zeclaw/

build-android-arm:
	GOOS=android GOARCH=arm CGO_ENABLED=0 go build -o ../frontend/assets/backend/zeclaw-backend-arm ./cmd/zeclaw/

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -f zeclaw zeclaw-backend-*
