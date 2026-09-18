// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

const base = (process.env.BASE_PATH || '/adr-repo-governance').replace(/\/$/, '') || '/';

export default defineConfig({
  base,
  site: 'https://averyfreeman.github.io',
  integrations: [
    starlight({
      title: 'adr-repo-governance',
      description: 'Git-native architecture decision record governance for teams.',
      disable404Route: true,
      customCss: ['./src/styles/custom.css'],
      social: [{ icon: 'github', label: 'GitHub', href: 'https://github.com/averyfreeman/adr-repo-governance' }],
      sidebar: [
        {
          label: 'Start here',
          items: [
            { label: 'Overview', link: '/' },
            { label: 'Workflow', slug: 'workflow' },
          ],
        },
        {
          label: 'Reference',
          items: [
            { label: 'Commands and model', slug: 'reference' },
          ],
        },
        {
          label: 'Contributing',
          items: [{ label: 'Development', slug: 'development' }],
        },
      ],
    }),
  ],
});
