#!/bin/sh
# Install git hooks for this repository.
set -e

HOOK=.git/hooks/pre-push
cat > "$HOOK" << 'EOF'
#!/bin/sh
# Run golangci-lint before every push.
set -e

if ! command -v golangci-lint >/dev/null 2>&1; then
  echo "⚠  golangci-lint not found — skipping lint (install: brew install golangci-lint)"
  exit 0
fi

echo "▸ Running golangci-lint..."
golangci-lint run --timeout=5m
echo "✓ Lint passed"
EOF
chmod +x "$HOOK"
echo "✓ pre-push hook installed"
