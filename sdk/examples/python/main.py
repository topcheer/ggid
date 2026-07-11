"""GGID Python SDK Quickstart — JWT login + verify in <20 lines.

Run:  GGID_URL=https://ggid.iot2.win python main.py
Local: GGID_URL=http://localhost:8080 python main.py
"""
import os

from ggid import GGIDClient


def main():
    url = os.environ.get("GGID_URL", "https://ggid.iot2.win")
    tenant_id = "00000000-0000-0000-0000-000000000001"

    # 1. Create client
    client = GGIDClient(url, tenant_id=tenant_id)

    # 2. Login to get JWT
    tokens = client.login("admin", "Admin@123456")
    access_token = tokens["access_token"]
    print(f"Login OK — access token: {len(access_token)} chars")

    # 3. Verify the token
    claims = client.verify_token(access_token)
    print(f"Verified — user: {claims.get('username')}, subject: {claims.get('sub')}")
    print("Quickstart complete!")


if __name__ == "__main__":
    main()
