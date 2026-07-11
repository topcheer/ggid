"""GGID Python SDK Quickstart — JWT login + verify in <20 lines.

Run:  GGID_URL=https://ggid.iot2.win python main.py
Local: GGID_URL=http://localhost:8080 python main.py

Requires: pip install ggid
"""
import asyncio
import os

from ggid import GGIDClient


async def main():
    url = os.environ.get("GGID_URL", "https://ggid.iot2.win")
    tenant_id = "00000000-0000-0000-0000-000000000001"

    # 1. Create client — jwks_url enables token verification
    client = GGIDClient(url, tenant_id=tenant_id, jwks_url=f"{url}/.well-known/jwks.json")

    # 2. Login to get JWT
    tokens = await client.login("sdk_test_user", "Xk9#Zm2!vQ7nRp")
    access_token = tokens["access_token"]
    print(f"Login OK — access token: {len(access_token)} chars")

    # 3. Verify the token
    claims = await client.verify_token(access_token)
    print(f"Verified — subject: {claims.sub}")
    print("Quickstart complete!")


if __name__ == "__main__":
    asyncio.run(main())
