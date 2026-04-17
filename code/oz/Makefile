BINARY := bin/oz

.PHONY: build install test lint clean run

build:
	go build -o $(BINARY) .

install: build
	cp $(BINARY) /usr/local/bin/oz

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -rf bin/

run: build
	./$(BINARY) $(ARGS)
