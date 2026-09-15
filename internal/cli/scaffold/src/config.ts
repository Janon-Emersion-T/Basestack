export const variants = {
  navbar: ['default', 'minimal'],
  hero: ['default', 'split'],
  features: ['default', 'list'],
  pricing: ['default', 'compact'],
  contact: ['default', 'centered'],
  footer: ['default', 'minimal'],
} as const;
export type Kind = keyof typeof variants;
export type Section = { id: string; type: Kind; title: string; text: string; variant?: string };
export type Page = { slug: string; title: string; sections: Section[] };
export type Config = { schemaVersion: 1; name: string; sections: Section[]; pages?: Page[] };
const namePattern = /^[a-z][a-z0-9-]{0,49}$/;

function object(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function unknownFields(value: Record<string, unknown>, allowed: string[], context: string) {
  if (Object.keys(value).some(key => !allowed.includes(key))) throw new Error(`${context}: unknown field.`);
}

function checkSections(value: unknown, context: string) {
  if (!Array.isArray(value)) throw new Error(`${context}: sections must be an array.`);
  const ids = new Set<string>();
  for (const section of value) {
    if (!object(section) || typeof section.id !== 'string' || !namePattern.test(section.id) || ids.has(section.id) || typeof section.type !== 'string' || !Object.prototype.hasOwnProperty.call(variants, section.type) || typeof section.title !== 'string' || !section.title.trim() || typeof section.text !== 'string') {
      throw new Error(`${context}: each section needs a unique valid id, supported type, non-empty title and text string.`);
    }
    unknownFields(section, ['id', 'type', 'title', 'text', 'variant'], context);
    if ('variant' in section) {
      const available: readonly string[] = variants[section.type as Kind];
      if (typeof section.variant !== 'string' || !available.includes(section.variant)) throw new Error(`${context}: unsupported ${section.type} variant.`);
    }
    ids.add(section.id);
  }
}

export function parseConfig(value: unknown): Config {
  if (!object(value) || value.schemaVersion !== 1 || typeof value.name !== 'string' || !namePattern.test(value.name)) {
    throw new Error('Expected schemaVersion 1 and a lowercase project name.');
  }
  unknownFields(value, ['schemaVersion', 'name', 'sections', 'pages'], 'Project');
  checkSections(value.sections, 'Home');
  if ('pages' in value) {
    if (!Array.isArray(value.pages)) throw new Error('Pages must be an array.');
    const slugs = new Set(['home']);
    for (const page of value.pages) {
      if (!object(page) || typeof page.slug !== 'string' || !namePattern.test(page.slug) || slugs.has(page.slug) || typeof page.title !== 'string' || !page.title.trim()) throw new Error('Each page needs a unique valid slug (home is reserved) and non-empty title.');
      unknownFields(page, ['slug', 'title', 'sections'], `Page ${page.slug}`);
      checkSections(page.sections, `Page ${page.slug}`);
      slugs.add(page.slug);
    }
  }
  return value as Config;
}

export function allPages(config: Config): Page[] {
  return [{ slug: 'home', title: 'Home', sections: config.sections }, ...(config.pages ?? [])];
}
