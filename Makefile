BINARY := autobot

.PHONY: darwin
darwin:
	mkdir -p release
	GOOS=darwin GOARCH=amd64 go build -o release/$(BINARY)-darwin-amd64 cmd/autobot/main.go

.PHONY: clean
clean:
	rm -rf release/*
