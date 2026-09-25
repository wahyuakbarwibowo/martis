# Martis TUI - Universal Makefile (macOS, Unix/Linux, & Windows)
BINARY_NAME=martis
MAIN_PACKAGE=.
INSTALL_DIR ?= /usr/local/bin
UPSTREAM_REMOTE ?= upstream
UPSTREAM_BRANCH ?= main
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-s -w -X main.version=$(VERSION)

.PHONY: all build release release-windows install uninstall update-upstream self-update run clean tidy fmt vet test help

all: build

## build: Compile release binary untuk sistem host saat ini
build:
	@echo "Building $(BINARY_NAME) ($(VERSION))..."
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Build complete: ./$(BINARY_NAME)"

## release: Compile release untuk semua platform (macOS, Linux, dan Windows)
release: clean
	@echo "Building optimized releases for macOS, Linux, and Windows..."
	@mkdir -p dist
	# macOS releases (Apple Silicon + Intel)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	# Unix / Linux releases
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-linux-arm64 $(MAIN_PACKAGE)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	# Windows releases (x64 + ARM64)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-windows-arm64.exe $(MAIN_PACKAGE)
	@echo ""
	@echo "All release binaries generated in dist/:"
	@ls -lh dist/

## release-windows: Compile khusus Windows binary (.exe)
release-windows:
	@mkdir -p dist
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-windows-arm64.exe $(MAIN_PACKAGE)
	@echo "Windows binaries compiled in dist/:"
	@ls -lh dist/*windows*

## install: Install binary martis ke PATH sistem lokal ($INSTALL_DIR)
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	@if [ -w $(INSTALL_DIR) ]; then \
		cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME); \
	else \
		sudo cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME); \
	fi
	@chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "✅ Installed! Jalankan perintah: $(BINARY_NAME)"

## uninstall: Hapus binary martis dari PATH sistem ($INSTALL_DIR)
uninstall:
	@echo "Uninstalling $(BINARY_NAME) from $(INSTALL_DIR)..."
	@if [ -w $(INSTALL_DIR)/$(BINARY_NAME) ]; then \
		rm -f $(INSTALL_DIR)/$(BINARY_NAME); \
	else \
		sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME); \
	fi
	@echo "✅ Uninstalled."

## update-upstream: Tarik update terbaru dari upstream git repo lalu rebuild & reinstall
update-upstream:
	@echo "⚡ Memeriksa pembaruan dari upstream ($(UPSTREAM_REMOTE)/$(UPSTREAM_BRANCH))..."
	@if git remote | grep -q "^$(UPSTREAM_REMOTE)$$"; then \
		git fetch $(UPSTREAM_REMOTE); \
		git merge --ff-only $(UPSTREAM_REMOTE)/$(UPSTREAM_BRANCH) || { echo "❌ Gagal fast-forward merge. Selesaikan conflict atau branch diverged."; exit 1; }; \
	else \
		echo "ℹ️  Remote '$(UPSTREAM_REMOTE)' tidak ditemukan. Menarik update dari origin..."; \
		git pull --ff-only origin $(UPSTREAM_BRANCH) || { echo "❌ Gagal git pull."; exit 1; }; \
	fi
	@echo "📦 Mengupdate dependensi Go..."
	@go mod tidy
	@echo "🔨 Rebuilding & menginstall versi terbaru..."
	@$(MAKE) install
	@echo "🚀 Update selesai dan versi terbaru siap digunakan!"

## self-update: Update binary terinstall via go install (tanpa perlu repo lokal)
self-update:
	@echo "⚡ Mengupdate binary martis langsung via Go toolchain..."
	go install -ldflags="$(LDFLAGS)" .
	@echo "✅ Binary terupdate di \$$GOPATH/bin/$(BINARY_NAME)"

## run: Menjalankan aplikasi langsung
run:
	go run $(MAIN_PACKAGE)

## tidy: Sinkronisasi dependensi Go
tidy:
	go mod tidy

## fmt: Format kode sumber Go
fmt:
	go fmt ./...

## vet: Analisa statis kode
vet:
	go vet ./...

## test: Menjalankan unit tests
test:
	go test -v -race ./...

## clean: Hapus binary dan direktori dist/
clean:
	@rm -rf $(BINARY_NAME) dist/
	@echo "Cleaned up."

## help: Tampilkan daftar target make
help:
	@echo "Martis TUI - Target Make yang tersedia:"
	@echo ""
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":"} {printf "  \033[38;5;39m%-18s\033[0m %s\n", $$1, $$2}'
	@echo ""
