import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { pathToFileURL } from 'node:url';

const { parseConfig } = await import(pathToFileURL(process.argv[2]).href);
const fixtures = JSON.parse(await readFile(new URL('../internal/cli/testdata/config-cases.json', import.meta.url), 'utf8'));
for (const fixture of fixtures) {
  if (fixture.valid) assert.doesNotThrow(() => parseConfig(fixture.config), fixture.name);
  else assert.throws(() => parseConfig(fixture.config), undefined, fixture.name);
}
console.log(`Frontend validation passed ${fixtures.length} shared configuration cases.`);
