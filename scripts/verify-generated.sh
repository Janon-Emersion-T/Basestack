#!/usr/bin/env bash
# End-to-end verification using a freshly generated project and its own PostgreSQL.
# Never enable shell tracing: the integration URL below is private.
set -euo pipefail
repo_dir="$(cd "$(dirname "$0")/.." && pwd)"
verify_dir="${1:-$(mktemp -d)}"
mkdir -p "$verify_dir"
verify_dir="$(cd "$verify_dir" && pwd)"
(cd "$repo_dir" && go build -o "$verify_dir/basestack" ./cmd/basestack)
cd "$verify_dir"
./basestack init smoke-app
cd smoke-app
cli="$verify_dir/basestack"
api_pid=""
frontend_pid=""
cleanup() {
  if [[ -n "$frontend_pid" ]]; then kill "$frontend_pid" 2>/dev/null || true; wait "$frontend_pid" 2>/dev/null || true; fi
  if [[ -n "$api_pid" ]]; then kill "$api_pid" 2>/dev/null || true; wait "$api_pid" 2>/dev/null || true; fi
  "$cli" services stop || true
}
trap cleanup EXIT
free_port() { node -e 'const s=require("node:net").createServer();s.listen(0,"127.0.0.1",()=>{console.log(s.address().port);s.close()})'; }
export BASESTACK_DATABASE_PORT="$(free_port)"
"$cli" services start
export BASESTACK_ENV=development
export BASESTACK_AUTH_DELIVERY=local
export BASESTACK_API_PORT="$(free_port)"
export BASESTACK_TEST_DATABASE_URL="$(node -e 'const fs=require("node:fs");const password=fs.readFileSync(".basestack/local.env","utf8").match(/^BASESTACK_DB_PASSWORD=(.+)$/m)[1];console.log(`postgres://basestack:${password}@127.0.0.1:${process.env.BASESTACK_DATABASE_PORT}/basestack?sslmode=disable`)')"
"$cli" db status
"$cli" db migrate
"$cli" migration new create_smoke_example
cat > basestack/migrations/000003_create_smoke_example.sql <<'SQL'
CREATE TABLE smoke_example (id integer PRIMARY KEY);
INSERT INTO smoke_example VALUES (1);
SQL
"$cli" db migrate
"$cli" db migrate
"$cli" services status
"$cli" page add about
"$cli" page add contact
"$cli" add navbar --page about --variant centered
"$cli" add hero --page about --variant split
"$cli" add features --variant cards
"$cli" add pricing --variant simple
"$cli" add contact --page contact --variant centered
"$cli" add footer --variant columns
"$cli" theme set modern
"$cli" theme set minimal
"$cli" theme set default
"$cli" check
# Root and generated tests both exercise real PostgreSQL in randomly named, isolated databases.
(cd "$repo_dir" && go test ./... && go test -race ./... && go vet ./...)
go test ./...
go test -race ./...
go vet ./...
npm install --no-fund --no-audit
npm test
npm run typecheck
"$cli" build
test -f dist/index.html
"$cli" auth status
export BASESTACK_AUTH_VERIFY_TTL_SECONDS=3
export BASESTACK_AUTH_RESET_TTL_SECONDS=3
"$cli" api > .basestack/api.log 2>&1 &
api_pid=$!
node --input-type=module <<'JS'
import assert from 'node:assert/strict';
const url=`http://127.0.0.1:${process.env.BASESTACK_API_PORT}/api/health`;
let ready=false;
for(let i=0;i<100;i++) {
  try {const r=await fetch(url);const h=await r.json();if(r.status===200&&h.status==='ok'&&h.service==='basestack'&&h.version==='0.3.1'&&h.database==='connected'){ready=true;break}} catch {}
  await new Promise(resolve=>setTimeout(resolve,100));
}
assert.ok(ready,'API did not become healthy');
assert.equal((await fetch(url,{headers:{Origin:'https://not-allowed.example'}})).status,403);
console.log('Live API health and CORS checks passed against PostgreSQL.');
JS
"$cli" dev > .basestack/frontend.log 2>&1 &
frontend_pid=$!
node --input-type=module <<'JS'
import {readFileSync} from 'node:fs';
import assert from 'node:assert/strict';
let ready=false;
for(let i=0;i<100;i++) {
  try {
    const log=readFileSync('.basestack/frontend.log','utf8');
    const address=log.match(/http:\/\/127\.0\.0\.1:\d+\//)?.[0];
    if(address){assert.match(await (await fetch(address+'about')).text(),/src\/main.tsx/);assert.match(log,/API: healthy; database: connected/);ready=true;break}
  } catch {}
  await new Promise(resolve=>setTimeout(resolve,100));
}
assert.ok(ready,'Frontend did not become ready with API status');
console.log('Frontend dev server and API status checks passed.');
JS
node "$repo_dir/scripts/verify-auth.mjs"
kill "$frontend_pid"; wait "$frontend_pid"; frontend_pid=""
kill "$api_pid"; wait "$api_pid"; api_pid=""
"$cli" services stop
"$cli" services status
"$cli" services start
"$cli" db status
printf 'Verification passed. Artifacts: %s/smoke-app/dist\n' "$verify_dir"
