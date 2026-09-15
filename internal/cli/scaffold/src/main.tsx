import React, { useEffect, useId } from 'react';
import { createRoot } from 'react-dom/client';
import rawConfig from '../basestack.json';
import { SectionView } from './sections';
import { allPages, parseConfig } from './config';
import type { Config } from './config';
import './styles.css';

function Project({ config }: { config: Config }) {
  const pages = allPages(config);
  const slug = new URLSearchParams(window.location.search).get('page') ?? 'home';
  const page = pages.find(item => item.slug === slug);
  const mainId = useId();
  useEffect(() => {
    document.title = `${page?.title ?? 'Page not found'} | ${config.name}`;
  }, [page?.title, config.name]);
  if (!page) return <main className="section"><h1>Page not found</h1><p>This page does not exist in your project.</p><a className="button" href="?page=home">Return home</a></main>;
  return <>
    <a className="skip-link" href={`#${mainId}`}>Skip to content</a>
    <main id={mainId} tabIndex={-1}>
      {page.sections.length === 0 && <section className="section"><h1>Your canvas is ready.</h1><p>Add a section with the BaseStack CLI.</p>{pages.length > 1 && <a href="?page=home">Return home</a>}</section>}
      {page.sections.map(section => <SectionView key={section.id} section={section} sections={page.sections} pages={pages} activePage={page.slug} />)}
    </main>
  </>;
}

function App() {
  try {
    return <Project config={parseConfig(rawConfig)} />;
  } catch (error) {
    return <main className="section" role="alert"><h1>Check your configuration</h1><p>{error instanceof Error ? error.message : 'Invalid configuration.'}</p><p>Edit basestack.json, then save to try again.</p></main>;
  }
}

createRoot(document.getElementById('root')!).render(<React.StrictMode><App /></React.StrictMode>);
