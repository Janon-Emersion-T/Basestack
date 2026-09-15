/// <reference types="vite/client" />
import React from 'react';
import { createRoot } from 'react-dom/client';
import rawConfig from '../basestack.json?raw';
import registry from './registry.json';
import { parseJSON, parseConfig, resolvePage } from './schema.mjs';
import { SectionView } from './sections';
import './styles.css';

function App() {
  try {
    const config = parseConfig(parseJSON(rawConfig), registry);
    const theme = registry.themes.find(theme => theme.name === config.theme)!;
    // Apply tokens to the document so the page background and native controls match.
    for (const [token, value] of Object.entries(theme.tokens)) document.documentElement.style.setProperty(`--${token}`, value);
    const page = resolvePage(config, window.location.pathname);
    document.title = page ? `${page.title} — ${config.name}` : `Page not found — ${config.name}`;
    if (!page) return <main className="section"><h1>Page not found</h1><p>Choose a page:</p><ul>{config.pages.map(p => <li key={p.id}><a href={p.path}>{p.title}</a></li>)}</ul></main>;
    const firstHero = page.sections.find(s => s.type === 'hero');
    return <>
      <a className="skip-link" href="#content">Skip to content</a>
      <main id="content" tabIndex={-1}>
        {!firstHero && <header className="section page-title"><h1>{page.title}</h1>{!page.sections.length && <p>Your canvas is ready. Add a section with the BaseStack CLI.</p>}</header>}
        {page.sections.map(section => <SectionView key={section.id} section={section} page={page} pages={config.pages} firstHero={section === firstHero} />)}
      </main>
    </>;
  } catch (error) {
    return <main className="section" role="alert"><h1>Check your configuration</h1><p>{error instanceof Error ? error.message : 'Invalid configuration.'}</p><p>Edit basestack.json, then save to try again.</p></main>;
  }
}
createRoot(document.getElementById('root')!).render(<React.StrictMode><App /></React.StrictMode>);
