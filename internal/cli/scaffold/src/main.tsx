import React from 'react';
import { createRoot } from 'react-dom/client';
import rawConfig from '../basestack.json';
import { SectionView, parseConfig } from './sections';
import './styles.css';

function App() {
  try {
    const config = parseConfig(rawConfig);
    return <>
      <a className="skip-link" href="#content">Skip to content</a>
      <main id="content" tabIndex={-1}>
        {config.sections.length === 0 && <section className="section"><h1>Your canvas is ready.</h1><p>Add a section with the BaseStack CLI.</p></section>}
        {config.sections.map(section => <SectionView key={section.id} section={section} sections={config.sections} />)}
      </main>
    </>;
  } catch (error) {
    return <main className="section" role="alert"><h1>Check your configuration</h1><p>{error instanceof Error ? error.message : 'Invalid configuration.'}</p><p>Edit basestack.json, then save to try again.</p></main>;
  }
}

createRoot(document.getElementById('root')!).render(<React.StrictMode><App /></React.StrictMode>);
