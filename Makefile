APP := download-organizer

.PHONY: test build clean

test:
	go test ./...

build:
	go build -trimpath -o $(APP) .

clean:
	rm -f $(APP)
