export interface DocPage {
	title: string;
	description: string;
	headings: string[];
	text: string;
	url: string;
}

const rawPages = import.meta.glob('../content/docs/**/*.{md,mdx}', {
	query: '?raw',
	import: 'default',
	eager: true,
}) as Record<string, string>;

export const pages: DocPage[] = Object.entries(rawPages)
	.map(([path, source]) => toPage(path, source))
	.filter((page): page is DocPage => page !== null);

function toPage(path: string, source: string): DocPage | null {
	if (path.endsWith('/404.md')) return null;

	const frontmatter = source.match(/^---\r?\n([\s\S]*?)\r?\n---/);
	const meta = frontmatter?.[1] ?? '';
	const title = field(meta, 'title');
	if (!title) return null;

	let body = source.slice(frontmatter ? frontmatter[0].length : 0);
	body = body.replace(/^import\s.+$/gm, '');
	body = body.replace(/<\/?[A-Za-z][^>]*>/g, ' ');
	const headings = [...body.matchAll(/^#{2,3}\s+(.+)$/gm)].map((match) =>
		match[1].replace(/[`*_]/g, '').trim(),
	);
	const text = body
		.replace(/```[\s\S]*?```/g, ' ')
		.replace(/`([^`]+)`/g, '$1')
		.replace(/^#{1,6}\s+/gm, '')
		.replace(/[*_>#|[\]()]/g, ' ')
		.replace(/\s+/g, ' ')
		.trim();

	return {
		title,
		description: field(meta, 'description'),
		headings,
		text,
		url: urlFor(path),
	};
}

function field(meta: string, name: string): string {
	const match = meta.match(new RegExp(`^${name}:\\s*(.+)$`, 'm'));
	return match?.[1]?.trim().replace(/^['"]|['"]$/g, '') ?? '';
}

function urlFor(path: string): string {
	let url = path.replace(/^.*content\/docs/, '').replace(/\.mdx?$/, '');
	if (url.endsWith('/index')) url = url.slice(0, -'/index'.length);
	if (url === '') return '/';
	return url.endsWith('/') ? url : `${url}/`;
}

export interface SearchHit {
	title: string;
	url: string;
	excerpt: string;
}

export function searchDocs(query: string, docs: DocPage[] = pages): SearchHit[] {
	const terms = query.trim().toLowerCase().split(/\s+/).filter(Boolean);
	if (terms.length === 0) return [];

	const hits: Array<SearchHit & { score: number }> = [];
	for (const doc of docs) {
		const title = doc.title.toLowerCase();
		const description = doc.description.toLowerCase();
		const headings = doc.headings.join(' ').toLowerCase();
		const body = doc.text.toLowerCase();
		if (!terms.every((term) => title.includes(term) || description.includes(term) || headings.includes(term) || body.includes(term))) {
			continue;
		}

		let score = 0;
		for (const term of terms) {
			if (title.includes(term)) score += 10;
			if (description.includes(term)) score += 4;
			if (headings.includes(term)) score += 3;
			if (body.includes(term)) score += 1;
		}
		hits.push({
			title: doc.title,
			url: doc.url,
			excerpt: excerptFor(doc, terms[0]),
			score,
		});
	}

	return hits
		.sort((a, b) => b.score - a.score || a.title.localeCompare(b.title))
		.slice(0, 8)
		.map(({ title, url, excerpt }) => ({ title, url, excerpt }));
}

function excerptFor(doc: DocPage, term: string): string {
	const source = doc.description || doc.text;
	const at = source.toLowerCase().indexOf(term);
	if (at < 0) return doc.description || doc.headings[0] || '';
	const start = Math.max(0, at - 48);
	const end = Math.min(source.length, at + term.length + 90);
	const slice = source.slice(start, end).trim();
	return `${start > 0 ? '…' : ''}${slice}${end < source.length ? '…' : ''}`;
}
