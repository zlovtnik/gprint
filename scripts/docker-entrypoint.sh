#!/bin/bash
set -e

# Decode wallet from base64 environment variable if provided
if [ -n "$WALLET_BASE64" ]; then
    echo "Decoding wallet from WALLET_BASE64..."
    mkdir -p /app/wallet
    echo "$WALLET_BASE64" | base64 -d > /tmp/wallet.zip
    unzip -q -o /tmp/wallet.zip -d /app/wallet
    rm /tmp/wallet.zip
    echo "Wallet decoded successfully"
    
    # Fix sqlnet.ora to use container path instead of local dev path
    if [ -f /app/wallet/sqlnet.ora ]; then
        sed -i 's|DIRECTORY="[^"]*"|DIRECTORY="/app/wallet"|g' /app/wallet/sqlnet.ora
        echo "Updated sqlnet.ora wallet path to /app/wallet"
        cat /app/wallet/sqlnet.ora
    fi
    
    # Create symlink so Oracle client finds tnsnames.ora
    # Oracle looks in $ORACLE_HOME/network/admin by default
    mkdir -p /opt/oracle/instantclient/network/admin
    ln -sf /app/wallet/* /opt/oracle/instantclient/network/admin/
    echo "Wallet files linked to Oracle network/admin"
fi

# Verify wallet exists
if [ ! -f /app/wallet/cwallet.sso ]; then
    echo "WARNING: Wallet file /app/wallet/cwallet.sso not found!"
    echo "Set WALLET_BASE64 env var or mount wallet at /app/wallet"
fi

# Execute the main application
exec "$@"
