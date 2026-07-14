#!/usr/bin/env node
// Simple helper script: login to GGID and print a JWT for testing.
// Usage: node get-token.js <username> <password>

const GGID_URL = process.env.GGID_URL || "https://ggid.iot2.win";
const GGID_TENANT_ID = process.env.GGID_TENANT_ID || "00000000-0000-0000-0000-000000000001";

async function main() {
  const [,, username, password] = process.argv;
  if (!username || !password) {
    console.error("Usage: node get-token.js <username> <password>");
    process.exit(1);
  }

  const resp = await fetch(`${GGID_URL}/api/v1/auth/login`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Tenant-ID": GGID_TENANT_ID,
    },
    body: JSON.stringify({ username, password }),
  });

  if (!resp.ok) {
    const text = await resp.text();
    console.error(`Login failed (${resp.status}): ${text}`);
    process.exit(1);
  }

  const data = await resp.json();
  console.log(data.access_token);
}

main().catch(console.error);
