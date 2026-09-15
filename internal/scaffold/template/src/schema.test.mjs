import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { parseJSON, parseConfig, resolvePage } from './schema.mjs';
const registry = JSON.parse(readFileSync(new URL('./registry.json', import.meta.url), 'utf8'));
const fresh = () => ({ schemaVersion: 2, name: 'test-app', theme: 'default', pages: [{ id: 'home', path: '/', title: 'Home', sections: registry.components.map(c => ({ id: c.type, type: c.type, variant: c.defaultVariant, props: structuredClone(c.defaultProps) })) }] });
test('all registry defaults, variants and themes validate', () => {
  for (const theme of registry.themes) for (const definition of registry.components) for (const variant of definition.variants) {
    const config = fresh(); config.theme = theme.name;
    config.pages[0].sections.find(s => s.type === definition.type).variant = variant;
    assert.equal(parseConfig(config, registry), config);
  }
});
test('routing matches exact paths, including nested paths and unknown routes', () => {
  const config = fresh(); config.pages.push({ id: 'about', path: '/company/about', title: 'About', sections: [] });
  parseConfig(config, registry);
  assert.equal(resolvePage(config, '/').id, 'home');
  assert.equal(resolvePage(config, '/company/about').id, 'about');
  assert.equal(resolvePage(config, '/missing'), undefined);
  assert.equal(resolvePage(config, '/company/about/'), undefined);
});
const invalid = {
  'unknown root': c => c.typo = true,
  'schema v1': c => c.schemaVersion = 1,
  'invalid name': c => c.name = '../app',
  'unknown theme': c => c.theme = 'oops',
  'null pages': c => c.pages = null,
  'empty pages': c => c.pages = [],
  'duplicate pages': c => c.pages.push(structuredClone(c.pages[0])),
  'duplicate path': c => c.pages.push({ ...structuredClone(c.pages[0]), id: 'about' }),
  'unknown page field': c => c.pages[0].oops = true,
  'empty page title': c => c.pages[0].title = '',
  'null sections': c => c.pages[0].sections = null,
  'duplicate section': c => c.pages[0].sections.push(c.pages[0].sections[0]),
  'reserved section ID': c => c.pages[0].sections[0].id = 'content',
  'unsafe section ID': c => c.pages[0].sections[0].id = '../x',
  'unknown section field': c => c.pages[0].sections[0].oops = true,
  'unknown type': c => c.pages[0].sections[0].type = 'oops',
  'unknown variant': c => c.pages[0].sections[0].variant = 'split',
  'missing variant': c => delete c.pages[0].sections[0].variant,
  'null props': c => c.pages[0].sections[0].props = null,
  'missing prop': c => delete c.pages[0].sections[0].props.subtitle,
  'null prop': c => c.pages[0].sections[0].props.title = null,
  'unknown prop': c => c.pages[0].sections[0].props.typo = true,
  'nested unknown': c => c.pages[0].sections[0].props.links = [{ label: 'X', href: '/', typo: true }],
  'nested missing': c => c.pages[0].sections[0].props.links = [{ label: 'X' }],
  'bad items': c => c.pages[0].sections[2].props.items = ['invalid'],
  'bad plans': c => c.pages[0].sections[3].props.plans = [{ title: 'Plan', price: 10, description: '' }],
  'bad email': c => c.pages[0].sections[4].props.email = 'invalid',
};
for (const [name, change] of Object.entries(invalid)) test(`rejects ${name}`, () => { const c = fresh(); change(c); assert.throws(() => parseConfig(c, registry)); });
for (const path of ['about', '//about', '/about/', '/../about', '/a//b', '/a?b', '/a#b', '/a%2fb', '/A', '/a b', '/a\\b']) test(`rejects path ${path}`, () => { const c = fresh(); c.pages[0].path = path; assert.throws(() => parseConfig(c, registry)); });
for (const href of ['javascript:alert(1)', '//evil.test', '/\\evil.test', 'data:text/html,hi', 'https://', '/a\nb']) test(`rejects unsafe link ${JSON.stringify(href)}`, () => { const c = fresh(); c.pages[0].sections[1].props.primaryButton.href = href; assert.throws(() => parseConfig(c, registry)); });
test('strict JSON rejects duplicate keys, corruption and trailing values', () => {
  for (const input of ['{"a":1,"a":2}', '{"a":[{"b":1,"b":2}]}', '{} {}', '{']) assert.throws(() => parseJSON(input));
  assert.deepEqual(parseJSON('{"a":[{"b":1},{"b":2}],"b":"text: \\"hi\\""}'), { a: [{ b: 1 }, { b: 2 }], b: 'text: "hi"' });
});
