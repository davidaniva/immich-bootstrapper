# Immich Importer Bootstrap

A minimal bootstrap binary for the Immich Google Photos Importer. This binary is personalized by the Immich server at download time with the user's server URL and setup token.

## How It Works

1. User clicks "Download Importer" in Immich web UI
2. Immich server takes a pre-compiled bootstrap binary and patches in:
   - The server URL
   - A temporary setup token (valid for 30 days)
3. User downloads and runs the patched binary (~2MB)
4. Bootstrap downloads the full Immich Importer app from GitHub Releases (~15MB)
5. Bootstrap launches the main app with `--server` and `--token` flags
6. Main app fetches config using the token and begins the import process

## Building

Requires Go 1.22+

```bash
# Build all platforms
make all

# Build specific platform
make darwin-arm64
make darwin-amd64
make linux-amd64
make windows-amd64

# Verify placeholder strings exist
make verify

# Install to Immich server
make install IMMICH_SERVER_DIR=/path/to/immich-fork/server
```

## Placeholder Strings

The binary contains two 128-character placeholder strings that are replaced at download time:

- `__IMMICH_SERVER_URL_PLACEHOLDER_...` - Replaced with the Immich server URL
- `__IMMICH_SETUP_TOKEN_PLACEHOLDER_...` - Replaced with the temporary setup token

The placeholders are padded with underscores to maintain a fixed length. The server null-terminates the actual values.

## Binary Patching

The Immich server patches the binary using simple string replacement:

```typescript
const paddedUrl = serverUrl.padEnd(128, '\0');
patched = binary.replace(placeholder, paddedUrl);
```

This approach avoids needing the Go toolchain on the server.

## Platforms

| Platform | File |
|----------|------|
| macOS ARM64 (Apple Silicon) | `bootstrap-darwin-arm64` |
| macOS AMD64 (Intel) | `bootstrap-darwin-amd64` |
| Linux AMD64 | `bootstrap-linux-amd64` |
| Windows AMD64 | `bootstrap-windows-amd64.exe` |

## Security

- The setup token is a one-time use token with a 30-day expiration
- After the main app fetches config, the token is invalidated
- The API key created has limited permissions (asset upload only)
- Users can revoke the API key from Immich after import completes
