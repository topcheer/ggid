/**
 * GGID Node.js SDK Quickstart — JWT login + verify in <20 lines.
 *
 * Run:  GGID_URL=https://ggid.iot2.win node index.js
 * Local: GGID_URL=http://localhost:8080 node index.js
 */

const { GGIDClient } = require('@ggid/node');

async function main() {
  const url = process.env.GGID_URL || 'https://ggid.iot2.win';
  const tenantId = '00000000-0000-0000-0000-000000000001';

  // 1. Create client — jwksUrl enables token verification
  const client = new GGIDClient({
    gatewayUrl: url,
    tenantId,
    jwksUrl: `${url}/.well-known/jwks.json`,
  });

  // 2. Login to get JWT
  const tokens = await client.login({
    username: 'sdk_test_user',
    password: 'Xk9#Zm2!vQ7nRp',
  });
  console.log(`Login OK — access token: ${tokens.access_token.length} chars`);

  // 3. Verify the token
  const claims = await client.verifyToken(tokens.access_token);
  console.log(`Verified — subject: ${claims.sub}`);
  console.log('Quickstart complete!');
}

main().catch(console.error);
