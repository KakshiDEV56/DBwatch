import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import { unified } from '@astrojs/markdown-remark';

const base = '/DBwatch';

// Markdown and MDX links are written as site-root paths (`/installation/`).
// GitHub Pages serves the site under /DBwatch/, so those hrefs have to pick
// up the base. Starlight's sidebar already does this; content links do not.
function remarkBaseLinks() {
	return (tree) => {
		walk(tree, (node) => {
			if (node.type === 'link' && typeof node.url === 'string') {
				node.url = withBase(node.url);
			}
			if (node.type === 'mdxJsxFlowElement' || node.type === 'mdxJsxTextElement') {
				for (const attr of node.attributes ?? []) {
					if (
						attr.type === 'mdxJsxAttribute' &&
						attr.name === 'href' &&
						typeof attr.value === 'string'
					) {
						attr.value = withBase(attr.value);
					}
				}
			}
		});
	};
}

function withBase(url) {
	if (!url.startsWith('/') || url.startsWith('//') || url === base || url.startsWith(`${base}/`)) {
		return url;
	}
	return base + url;
}

function walk(node, visit) {
	visit(node);
	for (const child of node.children ?? []) walk(child, visit);
}

// Project site: https://kakshidev56.github.io/DBwatch/
// `base` must match the GitHub repository name. GitHub Pages adds that
// prefix; the build output stays at dist/ and is not nested under DBwatch/.
export default defineConfig({
	site: 'https://kakshidev56.github.io',
	base,
	markdown: {
		processor: unified({
			remarkPlugins: [remarkBaseLinks],
		}),
	},
	integrations: [
		starlight({
			title: 'dbwatch',
			logo: {
				src: './src/assets/logo.png',
				alt: '',
			},
			description:
				'A terminal observability tool for PostgreSQL. See why a database is becoming unhealthy, from your own shell.',
			social: [
				{
					icon: 'github',
					label: 'GitHub',
					href: 'https://github.com/KakshiDEV56/DBwatch',
				},
			],
			editLink: {
				baseUrl: 'https://github.com/KakshiDEV56/DBwatch/edit/main/dbwatch-docs/',
			},
			lastUpdated: true,
			credits: false,
			tableOfContents: false,
			pagefind: false,
			favicon: '/favicon.png',
			components: {
				Search: './src/components/Search.astro',
			},
			customCss: ['./src/styles/custom.css'],
			expressiveCode: {
				themes: ['github-dark', 'github-light'],
			},
			head: [
				{
					tag: 'link',
					attrs: { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
				},
				{
					tag: 'link',
					attrs: { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' },
				},
				{
					tag: 'link',
					attrs: {
						rel: 'stylesheet',
						href: 'https://fonts.googleapis.com/css2?family=Geist:wght@400;500;600;700&family=Geist+Mono:wght@400;500&display=swap',
					},
				},
			],
			sidebar: [
				{
					label: 'Start',
					items: [
						{ label: 'Getting started', link: '/' },
						{ label: 'Installation', link: '/installation/' },
						{ label: 'Connection strings', link: '/connection-strings/' },
						{ label: 'Configuration', link: '/configuration/' },
						{ label: 'Docker', link: '/docker/' },
					],
				},
				{
					label: 'Monitoring',
					items: [
						{ label: 'Dashboard', link: '/monitoring/' },
						{ label: 'Connections', link: '/monitoring/connections/' },
						{ label: 'Cache hit ratio', link: '/monitoring/cache/' },
						{ label: 'Top queries', link: '/monitoring/queries/' },
						{ label: 'Locks and blocking', link: '/monitoring/locks/' },
						{ label: 'Long-running transactions', link: '/monitoring/transactions/' },
						{ label: 'Database size', link: '/monitoring/size/' },
						{ label: 'Capabilities', link: '/monitoring/capabilities/' },
						{ label: 'Errors and events', link: '/monitoring/errors/' },
						{ label: 'Health and load', link: '/monitoring/health/' },
					],
				},
				{
					label: 'Using dbwatch',
					items: [
						{ label: 'Databases', link: '/guides/databases/' },
						{ label: 'Keyboard and mouse', link: '/guides/keyboard/' },
						{ label: 'Demo environment', link: '/guides/demo/' },
					],
				},
				{
					label: 'Reference',
					items: [
						{ label: 'Thresholds', link: '/reference/thresholds/' },
						{ label: 'Limitations', link: '/reference/limitations/' },
						{ label: 'Changelog', link: '/changelog/' },
					],
				},
			],
		}),
	],
});
