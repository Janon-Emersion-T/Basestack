import type { Link, Page, Section } from './types';
function Links({ links, path }: { links: Link[]; path: string }) {
  return <div className="nav-links">{links.map((link, i) => <a key={i} href={link.href} aria-current={link.href === path ? 'page' : undefined}>{link.label}</a>)}</div>;
}
export function SectionView({ section: s, pages, page, firstHero }: { section: Section; pages: Page[]; page: Page; firstHero: boolean }) {
  const className = `section ${s.type} ${s.type}--${s.variant}`;
  const Heading = firstHero ? 'h1' : 'h2';
  switch (s.type) {
    case 'navbar': {
      const links = s.props.links.length ? s.props.links : pages.map(p => ({ label: p.title, href: p.path }));
      return <nav id={s.id} className={className} aria-label={s.props.title}><a className="brand" href={pages.find(p => p.path === '/')?.path ?? pages[0].path}>{s.props.title}</a><span className="nav-subtitle">{s.props.subtitle}</span><Links links={links} path={page.path} /></nav>;
    }
    case 'hero': return <section id={s.id} className={className}><div><Heading>{s.props.title}</Heading><p className="lead">{s.props.subtitle}</p>{s.props.primaryButton && <a className="button" href={s.props.primaryButton.href}>{s.props.primaryButton.label}</a>}</div><div className="composition" aria-hidden="true"><span>01 — Imagine</span><span>02 — Assemble</span><span>03 — Make it yours</span></div></section>;
    case 'features': return <section id={s.id} className={className}><h2>{s.props.title}</h2><p className="lead">{s.props.subtitle}</p><div className="feature-items">{s.props.items.map((item, i) => <article key={i} className="card"><h3>{item.title}</h3><p>{item.text}</p></article>)}</div></section>;
    case 'pricing': return <section id={s.id} className={className}><h2>{s.props.title}</h2><p className="lead">{s.props.subtitle}</p><div className="plans">{s.props.plans.map((plan, i) => <article key={i} className="price-card"><h3>{plan.title}</h3><strong className="price">{plan.price}</strong><p>{plan.description}</p></article>)}</div></section>;
    case 'contact': return <section id={s.id} className={className}><div><h2>{s.props.title}</h2><p className="lead">{s.props.subtitle}</p></div><a className="button" href={`mailto:${s.props.email}`}>Email {s.props.email}</a></section>;
    case 'footer': return <footer id={s.id} className={className}><strong>{s.props.title}</strong><p>{s.props.subtitle}</p><Links links={s.props.links} path={page.path} /><a href="#content">Back to top ↑</a></footer>;
  }
}
