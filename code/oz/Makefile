BINARY := bin/oz

.PHONY: build install install-completion-zsh install-completion-bash install-completion-fish install-completion-powershell test lint clean run

build:
	go build -o $(BINARY) .

install: build
	cp $(BINARY) /usr/local/bin/oz

install-completion-zsh: build
	mkdir -p "$(HOME)/.zsh/completions"
	./$(BINARY) completion zsh > "$(HOME)/.zsh/completions/_oz"

install-completion-bash: build
	mkdir -p "$(HOME)/.local/share/bash-completion/completions"
	./$(BINARY) completion bash > "$(HOME)/.local/share/bash-completion/completions/oz"

install-completion-fish: build
	mkdir -p "$(HOME)/.config/fish/completions"
	./$(BINARY) completion fish > "$(HOME)/.config/fish/completions/oz.fish"

install-completion-powershell: build
	./$(BINARY) completion powershell > oz.ps1

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -rf bin/

run: build
	./$(BINARY) $(ARGS)
