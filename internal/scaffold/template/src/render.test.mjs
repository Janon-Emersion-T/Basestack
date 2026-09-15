import test, { after } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, rmSync } from 'node:fs';
import { execFileSync } from 'node:child_process';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
execFileSync('node', ['node_modules/typescript/bin/tsc', 'src/sections.tsx', '--target', 'ES2022', '--module', 'ESNext', '--moduleResolution', 'bundler', '--jsx', 'react-jsx', '--strict', '--skipLibCheck', '--outDir', '.basestack-test'], { stdio: 'inherit' });
after(() => rmSync('.basestack-test', { recursive: true, force: true }));
const { SectionView } = await import('../.basestack-test/sections.js');
const registry = JSON.parse(readFileSync(new URL('./registry.json', import.meta.url), 'utf8'));
const pages = [{ id: 'home', path: '/', title: 'Home', sections: [] }, { id: 'about', path: '/about', title: 'About', sections: [] }];
const render = (definition, variant, firstHero = true) => renderToStaticMarkup(createElement(SectionView, { section: { id: definition.type, type: definition.type, variant, props: definition.defaultProps }, pages, page: pages[0], firstHero }));
for (const definition of registry.components) test(`${definition.type} renders both layouts and escapes content`, () => {
  const [a, b] = definition.variants.map(v => render(definition, v));
  assert.notEqual(a, b); assert.ok(a.includes(definition.defaultProps.title.replaceAll('&', '&amp;')));
  assert.ok(!a.includes('undefined'));
  const copy = structuredClone(definition); copy.defaultProps.title = '<script>alert(1)</script>';
  assert.ok(render(copy, 'default').includes('&lt;script&gt;'));
});
test('navigation links to every page and identifies current page', () => {
  const html = render(registry.components.find(c => c.type === 'navbar'), 'default');
  assert.match(html, /href="\/about"/); assert.match(html, /aria-current="page"/); assert.match(html, /<nav.*aria-label=/);
});
test('only the first hero uses h1', () => {
  const hero = registry.components.find(c => c.type === 'hero');
  assert.match(render(hero, 'split'), /<h1>/); assert.doesNotMatch(render(hero, 'split', false), /<h1>/);
});
