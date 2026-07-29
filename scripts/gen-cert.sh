#!/usr/bin/env bash
# Generate a self-signed code signing certificate for lol-telemetry Windows artifacts.
# Requires: openssl
# Usage: bash scripts/gen-cert.sh [output_dir]

set -euo pipefail

OUT_DIR="${1:-.}"
DAYS=730
SUBJ="/CN=lol-telemetry/O=Internal Distribution/C=BR"

echo "=== Generating self-signed code signing certificate ==="

# Generate private key + self-signed cert with Code Signing EKU
openssl req -x509 -newkey rsa:2048 -nodes \
  -keyout "$OUT_DIR/key.pem" \
  -out "$OUT_DIR/cert.pem" \
  -days "$DAYS" \
  -subj "$SUBJ" \
  -addext "extendedKeyUsage=codeSigning"

# Bundle into .pfx (for osslsigncode / signtool)
PFX_PASS="${PFX_PASS:-loltelemetry}"
openssl pkcs12 -export \
  -out "$OUT_DIR/lol-telemetry.pfx" \
  -inkey "$OUT_DIR/key.pem" \
  -in "$OUT_DIR/cert.pem" \
  -passout "pass:$PFX_PASS"

# Export public .cer (for testers to trust)
openssl x509 -in "$OUT_DIR/cert.pem" -outform DER \
  -out "$OUT_DIR/lol-telemetry.cer"

# Cleanup intermediate PEM files
rm -f "$OUT_DIR/key.pem" "$OUT_DIR/cert.pem"

echo "=== Done ==="
echo "Generated:"
echo "  $OUT_DIR/lol-telemetry.pfx  (signing - store as GitHub Secret)"
echo "  $OUT_DIR/lol-telemetry.cer  (public - commit or attach to releases)"
echo ""
echo "Next steps:"
echo "  1. base64 -i $OUT_DIR/lol-telemetry.pfx | pbcopy"
echo "  2. Add as GitHub Secret: CODE_SIGN_PFX"
echo "  3. Set GitHub Secret: CODE_SIGN_PASS (default: loltelemetry)"
echo "  4. Commit lol-telemetry.cer or attach to releases for testers"
