export const kinds = ['navbar', 'hero', 'features', 'pricing', 'contact', 'footer'] as const;
export type Kind = typeof kinds[number];
export type Section = { id: string; type: Kind; title: string; text: string };
export type Config = { schemaVersion: 1; name: string; sections: Section[] };
const namePattern = /^[a-z][a-z0-9-]{0,49}$/;

function object(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

export function parseConfig(value: unknown): Config {
  if (!object(value) || value.schemaVersion !== 1 || typeof value.name !== 'string' || !namePattern.test(value.name) || !Array.isArray(value.sections)) {
    throw new Error('Expected schemaVersion 1, a lowercase project name, and a sections array.');
  }
  if (Object.keys(value).some(key => !['schemaVersion', 'name', 'sections'].includes(key))) throw new Error('Unknown configuration field.');
  const ids = new Set<string>();
  for (const section of value.sections) {
    if (!object(section) || typeof section.id !== 'string' || !namePattern.test(section.id) || ids.has(section.id) || !kinds.includes(section.type as Kind) || typeof section.title !== 'string' || !section.title.trim() || typeof section.text !== 'string') {
      throw new Error('Each section needs a unique valid id, supported type, non-empty title and text string.');
    }
    if (Object.keys(section).some(key => !['id', 'type', 'title', 'text'].includes(key))) throw new Error(`Unknown field in section ${section.id}.`);
    ids.add(section.id);
  }
  return value as Config;
}

export function SectionView({ section: s, sections }: { section: Section; sections: Section[] }) {
  const target = sections.find(item => item.type === 'contact') ?? sections.find(item => item.type === 'features');
  switch (s.type) {
    case 'navbar': return <nav id={s.id} className="navbar" aria-label={s.title}>
      <a className="brand" href="#content"><span className="brand-mark" aria-hidden="true">B</span>{s.title}</a>
      <div className="nav-links">{sections.filter(item => ['features', 'pricing', 'contact'].includes(item.type)).map(item => <a key={item.id} href={`#${item.id}`}>{item.type}</a>)}</div>
    </nav>;
    case 'hero': return <section id={s.id} className="section hero">
      <span className="eyebrow"><span className="dot" /> Your idea. Your building blocks.</span>
      <h1>{s.title}</h1><p className="lead">{s.text}</p>
      {target && <a className="button" href={`#${target.id}`}>Explore more <span aria-hidden="true">↗</span></a>}
      <div className="hero-note">Designed to be yours, from the first block.</div>
      <div className="composition" aria-hidden="true"><div className="composition-line" /><div className="composition-card">01<span>Imagine</span></div><div className="composition-card">02<span>Assemble</span></div><div className="composition-card">03<span>Make it yours</span></div></div>
    </section>;
    case 'features': return <section id={s.id} className="section">
      <span className="eyebrow">Made for possibility</span><h2>{s.title}</h2><p className="lead">{s.text}</p>
      <div className="cards">{[['01', 'Start with a block', 'Choose reusable sections and assemble the page you need.'], ['02', 'Make it your own', 'Change content, colours and layout with editable source files.'], ['03', 'Keep control', 'Build and host your project wherever you choose.']].map(([n, title, text]) => <article className="card" key={n}><span className="card-number">{n}</span><h3>{title}</h3><p>{text}</p></article>)}</div>
    </section>;
    case 'pricing': return <section id={s.id} className="section pricing"><span className="eyebrow">Simple & transparent</span><h2>{s.title}</h2><p className="lead">{s.text}</p><article className="price-card"><span className="eyebrow">Your service</span><h3>Let’s find the right fit.</h3><p>Add your package, price and included services here.</p>{target && <a className="button" href={`#${target.id}`}>Learn more <span aria-hidden="true">↗</span></a>}</article></section>;
    case 'contact': return <section id={s.id} className="section contact"><span className="eyebrow">Start a conversation</span><h2>{s.title}</h2><p className="lead">{s.text}</p><a className="button" href="mailto:hello@example.com">Email us <span aria-hidden="true">↗</span></a></section>;
    case 'footer': return <footer id={s.id} className="footer"><strong>{s.title}</strong><p>{s.text}</p><a href="#content">Back to top ↑</a></footer>;
  }
}
