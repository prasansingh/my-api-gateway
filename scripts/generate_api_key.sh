#!/bin/bash
# Generates an API key and outputs the raw key, prefix, and hash
RAW_KEY=$(openssl rand -hex 32)
PREFIX=${RAW_KEY:0:8}
HASH=$(echo -n "$RAW_KEY" | sha256sum | awk '{print $1}')
echo "Raw key: $RAW_KEY"
echo "Prefix:  $PREFIX"
echo "Hash:    $HASH"
echo ""
echo "SQL:"
echo "INSERT INTO api_keys (key_hash, key_prefix, name) VALUES ('$HASH', '$PREFIX', 'my-key');"
