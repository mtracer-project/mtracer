#!/bin/sh
set -e

# Repository
REPO="mtracer-project/mtracer"

# Determine OS and Architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

if [ "$OS" = "Linux" ]; then
    OS_NAME="Linux"
    EXT="tar.zst"
elif [ "$OS" = "Darwin" ]; then
    OS_NAME="Darwin"
    EXT="zip"
else
    echo "This script is intended for Linux and macOS."
    echo "For Windows, please see the GitHub Releases page or use Winget/Scoop."
    exit 1
fi

# Map architecture to the format used by GoReleaser
if [ "$ARCH" = "x86_64" ]; then
    ARCH_NAME="x86_64"
elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
    ARCH_NAME="arm64"
else
    echo "Unsupported architecture: $ARCH"
    exit 1
fi

# Determine Version (use $1 if provided)
VERSION=$1

if [ -z "$VERSION" ]; then
    echo "Fetching the latest version..."
    LATEST_RELEASE=$(curl -sL "https://api.github.com/repos/$REPO/releases/latest")
    VERSION=$(echo "$LATEST_RELEASE" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$VERSION" ]; then
        echo "Failed to fetch the latest version. Check your internet connection or GitHub API limits."
        exit 1
    fi
    echo "Latest version is $VERSION"
else
    # Ensure it starts with a 'v' for the download URL if they just passed numbers
    case "$VERSION" in
        v*) ;;
        *) VERSION="v$VERSION" ;;
    esac
    echo "Using specified version: $VERSION"
fi

# The artifact name uses the version without the 'v' prefix
CLEAN_VERSION=$(echo "$VERSION" | sed 's/^v//')
FILENAME="mtracer_${CLEAN_VERSION}_${OS_NAME}_${ARCH_NAME}.${EXT}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"

# Download
TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

echo "Downloading $FILENAME from GitHub..."
curl -sL "$DOWNLOAD_URL" -o "$FILENAME"

# Extract
echo "Extracting $FILENAME..."
if [ "$EXT" = "zip" ]; then
    unzip -q "$FILENAME" || { echo "Extraction failed. Please make sure 'unzip' is installed."; exit 1; }
else
    # Attempt to extract with tar, check for zstd support
    if tar --version 2>/dev/null | grep -q 'GNU tar'; then
        tar --zstd -xf "$FILENAME" || { echo "Extraction failed. Please make sure 'zstd' is installed (e.g., sudo apt install zstd)."; exit 1; }
    else
        # Fallback to direct zstd | tar pipeline
        if command -v zstd >/dev/null 2>&1; then
            zstd -d "$FILENAME" --stdout | tar -xf -
        else
            echo "Error: 'zstd' command not found and tar does not support --zstd."
            echo "Please install zstd (e.g., sudo apt install zstd) and try again."
            exit 1
        fi
    fi
fi

# Install
INSTALL_DIR="/usr/local/bin"
echo "Installing 'mtracer' to $INSTALL_DIR (may require sudo)..."

# Ensure the extracted binary exists.
if [ -f "mtracer" ]; then
    sudo mv mtracer "$INSTALL_DIR/"
    sudo chmod +x "$INSTALL_DIR/mtracer"
elif [ -f "bin/mtracer" ]; then
    sudo mv bin/mtracer "$INSTALL_DIR/"
    sudo chmod +x "$INSTALL_DIR/mtracer"
else
    echo "Could not find 'mtracer' executable in the archive."
    exit 1
fi

# Clean up
cd - > /dev/null
rm -rf "$TMP_DIR"

echo "Installation complete!"
echo "Run 'mtracer --help' to verify."
