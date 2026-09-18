import config from './services.json';
import { createClient } from './client.mjs';

// Public settings only. An empty apiURL uses the application's own origin.
export const base = createClient(config);
