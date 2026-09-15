import type { Section, Page } from './config';

export function SectionView({ section: s, sections, pages, activePage }: { section: Section; sections: Section[]; pages: Page[]; activePage: string }) {
  const variant = s.variant ?? 'default';
  const className = (base: string) => `${base} variant-${variant}`;
  const target = sections.find(item => item.type === 'contact') ?? sections.find(item => item.type === 'features');
  switch (s.type) {
    case 'navbar': return <nav id={s.id} className={className("navbar")} aria-label={s.title}>
      <a className="brand" href="?page=home"><span className="brand-mark" aria-hidden="true">B</span>{s.title}</a>
      <div className="nav-links">{pages.length > 1 && pages.map(page => <a key={page.slug} href={`?page=${page.slug}`} aria-current={activePage === page.slug ? 'page' : undefined}>{page.title}</a>)}{sections.filter(item => ['features', 'pricing', 'contact'].includes(item.type)).map(item => <a key={item.id} href={`#${item.id}`}>{item.type}</a>)}</div>
    </nav>;
    case 'hero': return <section id={s.id} className={className("section hero")}>
      <div className="hero-copy"><span className="eyebrow"><span className="dot" /> Your idea. Your building blocks.</span>
      <h1>{s.title}</h1><p className="lead">{s.text}</p>
      {target && <a className="button" href={`#${target.id}`}>Explore more <span aria-hidden="true">↗</span></a>}
      <div className="hero-note">Designed to be yours, from the first block.</div></div>
      <div className="composition" aria-hidden="true"><div className="composition-line" /><div className="composition-card">01<span>Imagine</span></div><div className="composition-card">02<span>Assemble</span></div><div className="composition-card">03<span>Make it yours</span></div></div>
    </section>;
    case 'features': return <section id={s.id} className={className("section features")}>
      <span className="eyebrow">Made for possibility</span><h2>{s.title}</h2><p className="lead">{s.text}</p>
      <div className="cards">{[['01', 'Start with a block', 'Choose reusable sections and assemble the page you need.'], ['02', 'Make it your own', 'Change content, colours and layout with editable source files.'], ['03', 'Keep control', 'Build and host your project wherever you choose.']].map(([n, title, text]) => <article className="card" key={n}><span className="card-number">{n}</span><h3>{title}</h3><p>{text}</p></article>)}</div>
    </section>;
    case 'pricing': return <section id={s.id} className={className("section pricing")}><span className="eyebrow">Simple & transparent</span><h2>{s.title}</h2><p className="lead">{s.text}</p><article className="price-card"><span className="eyebrow">Your service</span><h3>Let’s find the right fit.</h3><p>Add your package, price and included services here.</p>{target && <a className="button" href={`#${target.id}`}>Learn more <span aria-hidden="true">↗</span></a>}</article></section>;
    case 'contact': return <section id={s.id} className={className("section contact")}><span className="eyebrow">Start a conversation</span><h2>{s.title}</h2><p className="lead">{s.text}</p><a className="button" href="mailto:hello@example.com">Email us <span aria-hidden="true">↗</span></a></section>;
    case 'footer': return <footer id={s.id} className={className("footer")}><strong>{s.title}</strong><p>{s.text}</p><a href="#">Back to top ↑</a></footer>;
  }
}
