export type Link = { label: string; href: string };
type Copy = { title: string; subtitle: string };
type Component<T extends string, V extends string, P> = { id: string; type: T; variant: 'default' | V; props: Copy & P };
export type Section =
  | Component<'navbar', 'centered', { links: Link[] }>
  | Component<'hero', 'split', { primaryButton?: Link }>
  | Component<'features', 'cards', { items: { title: string; text: string }[] }>
  | Component<'pricing', 'simple', { plans: { title: string; price: string; description: string }[] }>
  | Component<'contact', 'centered', { email: string }>
  | Component<'footer', 'columns', { links: Link[] }>;
export type Page = { id: string; path: string; title: string; sections: Section[] };
export type Config = { schemaVersion: 2; name: string; theme: string; pages: Page[] };
