#!/usr/bin/env python3
"""GGID Python SDK Quickstart — 5 minute integration.

Prerequisites:
  1. GGID running on localhost:8080
  2. pip install ggid-sdk

Run: python quickstart.py
"""
from ggid import GGIDClient

def main():
    # 1. Create client
    client = GGIDClient(base_url="http://localhost:8080")

    # 2. Login as admin
    token = client.login(email="admin@ggid.dev", password="Admin@123456")
    print("✓ Logged in as admin")

    # 3. List users
    users = client.list_users(access_token=token["access_token"])
    print(f"✓ Found {len(users)} users")
    for u in users[:3]:
        print(f"  - {u['email']} ({u.get('display_name', 'N/A')})")

    # 4. Create a new user
    new_user = client.create_user(
        access_token=token["access_token"],
        email="alice@company.com",
        display_name="Alice Chen",
        password="SecurePass#123",
    )
    print(f"✓ Created user: {new_user['email']} (id: {new_user['id']})")

    # 5. Delete the user (cleanup)
    client.delete_user(new_user["id"], access_token=token["access_token"])
    print(f"✓ Deleted user: {new_user['email']}")

if __name__ == "__main__":
    main()
