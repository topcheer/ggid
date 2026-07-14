#!/bin/sh
set -e

SO_PIN=${SOFTHSM2_SO_PIN:-1234}
USER_PIN=${SOFTHSM2_USER_PIN:-1234}
TOKEN_LABEL=${SOFTHSM2_TOKEN_LABEL:-ggid}
KEY_LABEL=${SOFTHSM2_KEY_LABEL:-ggid-signing-rsa}

# Initialize token in slot 0 if not already present
if ! softhsm2-util --show-slots 2>/dev/null | grep -q "label: $TOKEN_LABEL"; then
    echo "Initializing SoftHSM2 token: $TOKEN_LABEL"
    softhsm2-util --init-token --slot 0 \
        --label "$TOKEN_LABEL" \
        --so-pin "$SO_PIN" \
        --pin "$USER_PIN"
else
    echo "Token $TOKEN_LABEL already initialized"
fi

# Generate RSA key pair if not present
if ! pkcs11-tool --module /usr/lib/softhsm/libsofthsm2.so --list-objects --pin "$USER_PIN" 2>/dev/null | grep -q "label:.*$KEY_LABEL"; then
    echo "Generating RSA key pair: $KEY_LABEL"
    pkcs11-tool --module /usr/lib/softhsm/libsofthsm2.so \
        --login --pin "$USER_PIN" \
        --keypairgen --key-type RSA:2048 \
        --label "$KEY_LABEL" \
        --id "0001"
else
    echo "Key $KEY_LABEL already present"
fi

echo "SoftHSM2 ready with token '$TOKEN_LABEL' and key '$KEY_LABEL'"
echo "Library: /usr/lib/softhsm/libsofthsm2.so"
echo "User PIN: $USER_PIN"

# Keep container running to allow inspection
exec tail -f /dev/null
