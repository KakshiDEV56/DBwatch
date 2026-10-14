# dbwatch docs

Astro Starlight site for dbwatch. The header uses the dbwatch logo, the dark theme follows the terminal palette, and the accent colors are the logo blue and the green status light. The header theme control switches between that dark theme and a white theme.

```bash
npm install
npm run dev
```

Local pages are under `http://localhost:4321/DBwatch/` because the site is published as a GitHub project page.

`.github/workflows/docs.yml` builds this directory and deploys `dist/` to GitHub Pages. In the repository settings, set Pages to deploy from GitHub Actions. The public URL is `https://kakshidev56.github.io/DBwatch/`.
