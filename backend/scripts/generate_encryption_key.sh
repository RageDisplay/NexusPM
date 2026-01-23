#!/bin/bash

# Generate AES256 encryption key (32 bytes = 256 bits)
KEY=$(openssl rand -base64 32)

# Create or overwrite .env file
cat > .env << EOF
ENCRYPTION_KEY=$KEY
EOF

echo "✓ .env file created"
echo "✓ Encryption key: $KEY"
