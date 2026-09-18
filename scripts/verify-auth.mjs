// Explicit end-to-end DEVELOPMENT verification. Never print passwords or challenge secrets.
import assert from 'node:assert/strict';
import { readFileSync, readdirSync } from 'node:fs';
const base=`http://127.0.0.1:${process.env.BASESTACK_API_PORT}`;
const password='BaseStack integration test passphrase';
const replacement='Another integration test passphrase';
const secrets=[password,replacement];
const wait=ms=>new Promise(resolve=>setTimeout(resolve,ms));
async function post(path,body){
 const r=await fetch(base+path,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
 const text=await r.text();for(const secret of secrets)assert.ok(!text.includes(secret),'sensitive value in API response');
 assert.ok(!text.includes('password_hash')&&!text.includes('$argon2')&&!text.includes('digest'),'internal auth data in response');
 return {status:r.status,data:JSON.parse(text)};
}
function challenge(email,purpose){
 const messages=readdirSync('.basestack/auth-outbox').map(name=>JSON.parse(readFileSync(`.basestack/auth-outbox/${name}`,'utf8'))).filter(m=>m.email===email&&m.purpose===purpose).sort((a,b)=>a.expiresAt.localeCompare(b.expiresAt));
 assert.ok(messages.length,'missing development delivery');const token=messages.at(-1).token;secrets.push(token);return token;
}
const signup=await post('/api/auth/signup',{email:'live-auth@example.com',password});assert.equal(signup.status,201);assert.equal(signup.data.data.user.status,'unverified');
assert.equal((await post('/api/auth/login',{email:'live-auth@example.com',password})).status,401);
const verification=challenge('live-auth@example.com','verify_email');
assert.equal((await post('/api/auth/verify-email',{token:verification})).status,200);
assert.equal((await post('/api/auth/verify-email',{token:verification})).status,400);
assert.equal((await post('/api/auth/verify-email',{token:'not-a-token'})).status,400);
const login=await post('/api/auth/login',{email:'LIVE-AUTH@example.com',password});assert.equal(login.status,200);assert.equal(login.data.data.credentialsVerified,true);assert.equal(login.data.data.sessionIssued,false);
assert.equal((await post('/api/auth/signup',{email:'LIVE-AUTH@example.com',password})).status,409);
const existing=await post('/api/auth/password/forgot',{email:'live-auth@example.com'});
const absent=await post('/api/auth/password/forgot',{email:'absent-auth@example.com'});assert.equal(existing.status,202);assert.deepEqual(existing,absent);
const reset=challenge('live-auth@example.com','reset_password');
assert.equal((await post('/api/auth/password/reset',{token:reset,password:replacement})).status,200);
assert.equal((await post('/api/auth/password/reset',{token:reset,password})).status,400);
assert.equal((await post('/api/auth/login',{email:'live-auth@example.com',password})).status,401);
assert.equal((await post('/api/auth/login',{email:'live-auth@example.com',password:replacement})).status,200);
// The parent script explicitly sets short development-only test TTLs for these expiry checks.
assert.equal((await post('/api/auth/signup',{email:'expired-auth@example.com',password})).status,201);
const expiredVerification=challenge('expired-auth@example.com','verify_email');
assert.equal((await post('/api/auth/password/forgot',{email:'live-auth@example.com'})).status,202);
const expiredReset=challenge('live-auth@example.com','reset_password');
await wait(3500);
assert.equal((await post('/api/auth/verify-email',{token:expiredVerification})).status,400);
assert.equal((await post('/api/auth/password/reset',{token:expiredReset,password})).status,400);
assert.equal((await post('/api/auth/verify-email/request',{email:'expired-auth@example.com'})).status,202);
const resent=challenge('expired-auth@example.com','verify_email');assert.notEqual(resent,expiredVerification);assert.equal((await post('/api/auth/verify-email',{token:resent})).status,200);
for(const file of ['.basestack/api.log','.basestack/frontend.log']){
 const log=readFileSync(file,'utf8');for(const secret of secrets)assert.ok(!log.includes(secret),'sensitive value in runtime log');
 assert.ok(!log.includes('$argon2')&&!log.includes('password_hash'),'password metadata in runtime log');
}
console.log('Live Auth signup, verification, login, reset, expiry/reuse and response/log secrecy checks passed.');
