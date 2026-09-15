import type { Config, Page } from './types';
export function parseJSON(text: string): unknown;
export function parseConfig(value: unknown, registry: unknown): Config;
export function resolvePage(config: Config, pathname: string): Page | undefined;
