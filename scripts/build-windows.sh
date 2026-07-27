#!/usr/bin/env sh
# Build and verify the Windows artifacts for lol-telemetry.

set -e

make build-windows

verify_pe() {
    FILE="$1"
    if [ ! -f "$FILE" ]; then
        echo "error: expected artifact not found: $FILE" >&2
        exit 1
    fi

    SIZE=$(stat -c%s "$FILE" 2>/dev/null || stat -f%z "$FILE" 2>/dev/null)

    # Verify the file starts with the Windows PE executable magic bytes (MZ).
    PE_MAGIC=$(head -c 2 "$FILE" | xxd -p 2>/dev/null || printf '%s' "unknown")
    if [ "$PE_MAGIC" != "4d5a" ]; then
        echo "error: $FILE does not appear to be a Windows PE executable" >&2
        exit 1
    fi

    echo "Windows artifact OK: $FILE ($SIZE bytes)"
}

verify_pe "dist/lol-daemon.exe"
verify_pe "dist/lol-cli.exe"
