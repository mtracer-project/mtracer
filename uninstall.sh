#!/bin/sh
set -e

INSTALL_DIR="/usr/local/bin"
MTRACER_BIN="$INSTALL_DIR/mtracer"

if [ -f "$MTRACER_BIN" ]; then
    echo "Removing mtracer from $MTRACER_BIN (may require sudo)..."
    sudo rm -f "$MTRACER_BIN"
    echo "mtracer has been successfully uninstalled."
else
    # Fallback: Try to find it in PATH if it was installed elsewhere
    MTRACER_PATH=$(command -v mtracer || true)
    
    if [ -n "$MTRACER_PATH" ]; then
        echo "Found mtracer at $MTRACER_PATH."
        echo "Removing mtracer (may require sudo)..."
        sudo rm -f "$MTRACER_PATH"
        echo "mtracer has been successfully uninstalled."
    else
        echo "mtracer does not seem to be installed in $INSTALL_DIR or anywhere in your PATH."
        echo "Nothing to uninstall."
    fi
fi
