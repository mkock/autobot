BINARY := autobot

.PHONY: darwin
darwin:
	mkdir -p release
	GOOS=darwin GOARCH=amd64 go build -o release/$(BINARY)-darwin-amd64 cmd/autobot/autobot.go

.PHONY: clean
clean:
	rm -rf release/*

.PHONY: install	
install:
	GOOS=darwin GOARCH=amd64 go install cmd/autobot/autobot.go
