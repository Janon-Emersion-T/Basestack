// This validator consumes the same portable registry as the Go CLI.
export function parseJSON(text) {
  const value = JSON.parse(text);
  const stack = [];
  for (const match of text.matchAll(/"(?:\\.|[^"\\])*"|[{}\[\],:]/g)) {
    const token = match[0];
    if (token === '{' || token === '[') stack.push({ object: token === '{', key: true, keys: new Set() });
    else if (token === '}' || token === ']') stack.pop();
    else {
      const frame = stack.at(-1);
      if (frame?.object && token === ',') frame.key = true;
      else if (frame?.object && frame.key && token.startsWith('"')) {
        const key = JSON.parse(token);
        if (frame.keys.has(key)) throw new Error(`Duplicate JSON field ${key}.`);
        frame.keys.add(key); frame.key = false;
      }
    }
  }
  return value;
}
const namePattern = /^[a-z][a-z0-9-]{0,49}$/;
const pathPattern = /^\/(?:[a-z0-9][a-z0-9_-]*(?:\/[a-z0-9][a-z0-9_-]*)*)?$/;
const emailPattern = /^[A-Za-z0-9.!#$%&'*+/=?^_`{|}~-]+@[A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)+$/;
const object = value => typeof value === 'object' && value !== null && !Array.isArray(value);
function fields(value, keys, path) {
  if (!object(value)) throw new Error(`${path} must be an object.`);
  for (const key of Object.keys(value)) if (!keys.includes(key)) throw new Error(`Unknown field ${path}.${key}.`);
}
function text(value, path) {
  if (typeof value !== 'string' || !value.trim()) throw new Error(`${path} must be a non-empty string.`);
}
function safeLink(value) {
  if (!value || /[\\\s\u0000-\u001f\u007f]/.test(value)) return false;
  if (value.startsWith('#')) return value.length > 1;
  if (value.startsWith('/')) return !value.startsWith('//');
  if (value.startsWith('mailto:')) return emailPattern.test(value.slice(7));
  if (!/^https?:\/\//.test(value)) return false;
  try { return Boolean(new URL(value).hostname); } catch { return false; }
}
function validateProps(value, rule, path) {
  if (rule.type === 'object') {
    fields(value, Object.keys(rule.fields), path);
    for (const [key, child] of Object.entries(rule.fields)) {
      if (!(key in value)) { if (child.required) throw new Error(`${path}.${key} is required.`); }
      else validateProps(value[key], child, `${path}.${key}`);
    }
  } else if (rule.type === 'array') {
    if (!Array.isArray(value)) throw new Error(`${path} must be an array.`);
    value.forEach((item, i) => validateProps(item, rule.items, `${path}[${i}]`));
  } else if (rule.type === 'string') {
    if (typeof value !== 'string') throw new Error(`${path} must be a string.`);
    if (rule.nonEmpty) text(value, path);
    if (rule.format === 'link' && !safeLink(value)) throw new Error(`${path} must be a safe link.`);
    if (rule.format === 'email' && !emailPattern.test(value)) throw new Error(`${path} must be an email address.`);
  } else throw new Error(`Unknown registry rule ${rule.type}.`);
}
export function parseConfig(value, registry) {
  fields(value, ['schemaVersion', 'name', 'theme', 'pages'], 'config');
  if (value.schemaVersion !== 2) throw new Error('Expected schemaVersion 2. V1 projects require manual migration.');
  if (typeof value.name !== 'string' || !namePattern.test(value.name)) throw new Error('Invalid project name.');
  if (!registry.themes.some(theme => theme.name === value.theme)) throw new Error('Unknown theme.');
  if (!Array.isArray(value.pages) || !value.pages.length) throw new Error('pages must be a non-empty array.');
  const ids = new Set(), paths = new Set();
  for (const page of value.pages) {
    fields(page, ['id', 'path', 'title', 'sections'], 'page');
    if (typeof page.id !== 'string' || !namePattern.test(page.id) || ids.has(page.id)) throw new Error('Invalid or duplicate page id.');
    if (typeof page.path !== 'string' || !pathPattern.test(page.path) || paths.has(page.path)) throw new Error('Invalid or duplicate page path.');
    ids.add(page.id); paths.add(page.path); text(page.title, 'page.title');
    if (!Array.isArray(page.sections)) throw new Error('sections must be an array.');
    const sections = new Set(['content']);
    for (const section of page.sections) {
      fields(section, ['id', 'type', 'variant', 'props'], 'section');
      if (typeof section.id !== 'string' || !namePattern.test(section.id) || sections.has(section.id)) throw new Error('Invalid, reserved or duplicate section id.');
      sections.add(section.id);
      const definition = registry.components.find(component => component.type === section.type);
      if (!definition) throw new Error(`Unknown section type ${section.type}.`);
      if (!definition.variants.includes(section.variant)) throw new Error(`Unknown ${section.type} variant ${section.variant}.`);
      validateProps(section.props, definition.props, `section ${section.id} props`);
    }
  }
  return value;
}
export function resolvePage(config, pathname) { return config.pages.find(page => page.path === pathname); }
