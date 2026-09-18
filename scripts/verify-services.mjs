// Real generated-app verification. Never print credentials or subprocess error buffers.
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { readFileSync, readdirSync, renameSync } from 'node:fs';
const cli = process.env.BASESTACK_VERIFY_CLI;
const origin = `http://127.0.0.1:${process.env.BASESTACK_API_PORT}`;
const privateValues = ['Services integration secret passphrase', 'Private environment verification value'];
function command(args, input, env = {}) {
  try { return execFileSync(cli, args, { input, encoding: 'utf8', env: { ...process.env, ...env }, stdio: ['pipe', 'pipe', 'pipe'] }); }
  catch { throw new Error(`CLI verification failed for ${args[0]} ${args[1]}`); }
}
const request = (path, method = 'GET', body, token) => fetch(origin + path, {
  method, headers: { ...(token ? { Authorization: `Bearer ${token}` } : {}), ...(body === undefined || body instanceof Blob ? {} : { 'Content-Type': 'application/json' }) },
  body: body === undefined ? undefined : body instanceof Blob ? body : JSON.stringify(body),
});
assert.match(command(['env', 'list']), /development\ntest\nproduction/);
command(['env', 'set', 'test', 'PRIVATE_VERIFY_VALUE'], privateValues[1]);
const shown = command(['env', 'show', 'test']);
assert.ok(shown.includes('[redacted]') && !shown.includes(privateValues[1]));
command(['env', 'unset', 'test', 'PRIVATE_VERIFY_VALUE']);
assert.ok(!command(['env', 'show', 'test']).includes('PRIVATE_VERIFY_VALUE'));
assert.match(command(['services', 'list']), /storage: true/);
assert.match(command(['functions', 'list']), /hello/);
assert.match(command(['functions', 'inspect', 'whoami']), /profile.read/);
assert.match(command(['functions', 'run', 'hello'], '{}'), /Hello from BaseStack/);
assert.equal((await request('/api/functions/hello', 'POST', {})).status, 200);
assert.equal((await request('/api/functions/whoami', 'POST', {})).status, 401);

const email = 'services-smoke@example.com';
const signup = await request('/api/auth/signup', 'POST', { email, password: privateValues[0] });
assert.equal(signup.status, 201);
const user = (await signup.json()).data.user;
const message = readdirSync('.basestack/auth-outbox').map(n => JSON.parse(readFileSync(`.basestack/auth-outbox/${n}`, 'utf8'))).find(m => m.email === email && m.purpose === 'verify_email');
assert.ok(message);
privateValues.push(message.token);
assert.equal((await request('/api/auth/verify-email', 'POST', { token: message.token })).status, 200);
const login = await request('/api/auth/login', 'POST', { email, password: privateValues[0] });
assert.equal(login.status, 200);
const token = (await login.json()).data.session.token;
privateValues.push(token);
assert.equal((await request('/api/functions/whoami', 'POST', {}, token)).status, 403);
command(['roles', 'create', 'smoke-reader']);
command(['roles', 'grant', 'smoke-reader', 'profile.read']);
command(['roles', 'assign', user.id, 'smoke-reader']);
command(['roles', 'check', user.id, 'profile.read']);
const identity = await request('/api/functions/whoami', 'POST', {}, token);
assert.equal(identity.status, 200); assert.equal((await identity.json()).data.id, user.id);
assert.match(command(['functions', 'run', 'whoami'], '{}', { BASESTACK_FUNCTION_TOKEN: token }), new RegExp(user.id));

command(['storage', 'create-bucket', 'files']);
assert.equal((await request('/api/storage/files', 'GET', undefined, token)).status, 403);
command(['roles', 'grant', 'smoke-reader', 'storage.files.read']);
command(['roles', 'grant', 'smoke-reader', 'storage.files.write']);
const upload = await request('/api/storage/files', 'POST', new Blob(['stored object']), token);
assert.equal(upload.status, 200);
const object = (await upload.json()).data;
assert.equal(object.size, 13);
const downloaded = await request(`/api/storage/files/${object.id}`, 'GET', undefined, token);
assert.equal(downloaded.status, 200); assert.equal(await downloaded.text(), 'stored object');
assert.equal((await request(`/api/storage/files/${object.id}`)).status, 401);
assert.equal((await request('/api/storage/files/%2Fetc%2Fpasswd', 'GET', undefined, token)).status, 400);
assert.equal((await request(`/api/storage/files/${object.id}`, 'DELETE', undefined, token)).status, 200);
assert.equal((await request(`/api/storage/files/${object.id}`, 'GET', undefined, token)).status, 404);
const local = JSON.parse(command(['storage', 'put', 'files'], 'CLI bytes'));
assert.equal(command(['storage', 'get', 'files', local.id]), 'CLI bytes');
assert.match(command(['storage', 'list', 'files']), new RegExp(local.id));
command(['storage', 'delete', 'files', local.id]);
command(['roles', 'revoke', 'smoke-reader', 'profile.read']);
assert.equal((await request('/api/functions/whoami', 'POST', {}, token)).status, 403);
assert.equal((await request('/api/auth/logout', 'POST', undefined, token)).status, 200);

// Public frontend-only v2 projects retain their composition workflow.
renameSync('basestack/services.json', '.basestack/services.saved.json');
try { command(['check']); command(['build']); }
finally { renameSync('.basestack/services.saved.json', 'basestack/services.json'); }
for (const file of ['.basestack/api.log', '.basestack/frontend.log']) {
  const log = readFileSync(file, 'utf8');
  for (const secret of privateValues) assert.ok(!log.includes(secret), 'private value in runtime log');
}
console.log('Environment, RBAC, storage, functions, sessions and frontend-only compatibility checks passed.');
