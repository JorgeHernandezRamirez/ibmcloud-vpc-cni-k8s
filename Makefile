# Nombre del binario
BINARY_NAME=ibmcloud-vpc-cni-k8s

# Go build flags
BUILD_FLAGS=-ldflags "-s -w"

# Directorio de instalación
INSTALL_DIR?=/usr/local/bin

.PHONY: all build install clean

# Target por defecto
all: build

# Compilar el binario
build:
	@echo "📦 Building $(BINARY_NAME)..."
	go build $(BUILD_FLAGS) -o $(BINARY_NAME) main.go

# Instalar el binario en $(INSTALL_DIR)
install: build
	@echo "🚀 Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	cp $(BINARY_NAME) $(INSTALL_DIR)

# Limpiar binarios
clean:
	@echo "🧹 Cleaning..."
	rm -f $(BINARY_NAME)
