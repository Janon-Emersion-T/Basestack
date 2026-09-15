import { readFileSync } from 'node:fs';
import { parseJSON, parseConfig } from './src/schema.mjs';
const read = path => parseJSON(readFileSync(new URL(path, import.meta.url), 'utf8'));
parseConfig(read('./basestack.json'), read('./src/registry.json'));
console.log('Configuration is valid.');
