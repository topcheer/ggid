/**
 * GGID AI Agent Identity — Node.js Quickstart
 * 
 * Complete flow: register → exchange token → verify
 * 
 * Prerequisites:
 *   npm install axios
 * 
 * Usage:
 *   GGID_API=http://localhost:8080 \
 *   GGID_TENANT=00000000-0000-0000-0000-000000000001 \
 *   SUBJECT_TOKEN=<your-jwt> \
 *   npx tsx agent-example.ts
 */

import axios from "axios";

const API = process.env.GGID_API || "http://localhost:8080";
const TENANT = process.env.GGID_TENANT || "00000000-0000-0000-0000-000000000001";
const SUBJECT_TOKEN = process.env.SUBJECT_TOKEN || "";

if (!SUBJECT_TOKEN) {
  console.error("Error: Set SUBJECT_TOKEN env var to a valid user JWT");
  process.exit(1);
}

const client = axios.create({
  baseURL: API,
  headers: {
    "Content-Type": "application/json",
    "X-Tenant-ID": TENANT,
  },
});

async function main() {
  // Step 1: Register an AI Agent
  console.log("1. Registering agent...");
  const registerResp = await client.post("/api/v1/agents/register", {
    name: "my-coding-assistant",
    type: "coding-assistant",
    owner_user_id: "00000000-0000-0000-0000-000000000001",
    allowed_scopes: ["read:repos", "write:repos"],
    max_delegation_depth: 3,
    allowed_mcp_servers: ["github.com/mcp"],
  });
  const agent = registerResp.data;
  console.log("   Agent ID:", agent.agent_id);
  console.log("   Client ID:", agent.client_id);
  console.log("   Client Secret:", agent.client_secret ? "(revealed once)" : "N/A");

  // Step 2: Exchange subject token for agent token
  console.log("\n2. Exchanging token...");
  const tokenResp = await client.post("/api/v1/agents/token", {
    subject_token: SUBJECT_TOKEN,
    agent_id: agent.agent_id,
    scope: "read:repos write:repos",
    audience: "api.ggid.dev",
  });
  const agentToken = tokenResp.data.access_token || tokenResp.data.token;
  console.log("   Agent token (first 50):", agentToken?.slice(0, 50) + "...");
  console.log("   Expires in:", tokenResp.data.expires_in, "seconds");

  // Step 3: Verify the agent token
  console.log("\n3. Verifying agent token...");
  const verifyResp = await client.post("/api/v1/agents/verify", {
    token: agentToken,
  });
  const claims = verifyResp.data;
  console.log("   Active:", claims.active);
  console.log("   Agent ID:", claims.agent_id);
  console.log("   Scope:", claims.scope);
  console.log("   Delegation depth:", claims.max_delegation_depth);

  console.log("\n✓ Complete! Agent identity workflow verified.");
}

main().catch((err) => {
  console.error("Error:", err.response?.data || err.message);
  process.exit(1);
});
