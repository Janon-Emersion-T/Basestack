import test from 'node:test';
import assert from 'node:assert/strict';
import { createClient } from './client.mjs';

test('client keeps tokens in memory, sends bearer authorization and clears on logout', async () => {
  const calls = [];
  const client = createClient({ schemaVersion: 1, apiURL: '' }, async (path, options) => {
    calls.push({ path, options });
    if (path.endsWith('/login')) return Response.json({ data: { sessionIssued: true, session: { token: 'private' }, user: { id: 'u' } } });
    return Response.json({ data: { user: { id: 'u' } } });
  });
  assert.deepEqual(await client.auth.login('user@example.com', 'passphrase'), { id: 'u' });
  await client.auth.current();
  assert.equal(calls.at(-1).options.headers.Authorization, 'Bearer private');
  assert.equal(calls.at(-1).options.credentials, 'omit');
  assert.equal(calls.at(-1).options.redirect, 'error');
  await client.auth.logout();
  await client.functions.run('hello', { test: true });
  assert.equal(calls.at(-1).options.headers.Authorization, undefined);
  assert.equal(calls.at(-1).path, '/api/functions/hello');
  assert.throws(() => client.functions.run('../private'));
});
test('errors do not echo server bodies and configuration rejects credential URLs', async () => {
  for (const apiURL of ['https://user:secret@example.com', 'javascript:alert(1)', 'https://example.com?secret=value']) assert.throws(() => createClient({ schemaVersion: 1, apiURL }));
  const client = createClient({ schemaVersion: 1, apiURL: 'https://example.com' }, async () => new Response('PRIVATE_DRIVER_ERROR', { status: 503 }));
  await assert.rejects(client.functions.run('hello'), { message: 'BaseStack request failed (503).' });
});
