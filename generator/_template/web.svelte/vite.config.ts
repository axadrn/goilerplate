import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig, loadEnv } from 'vite';

// The paths the Go server answers. In development Vite forwards them to
// GO_ORIGIN, the Go server task dev starts, so the browser sees one origin
// like in production.
const serverPaths = '^/(api|mcp|webhooks|healthz|auth/google|auth/github)(/|$)';

export default defineConfig(({ mode }) => {
	const { GO_ORIGIN = 'http://localhost:8090' } = loadEnv(mode, '.', 'GO_');

	return {
		plugins: [
			tailwindcss(),
			sveltekit({
				compilerOptions: {
					runes: ({ filename }) =>
						filename.split(/[/\\]/).includes('node_modules') ? undefined : true
				},
				// The Go binary embeds build/ and answers every page path with
				// index.html, so the app is a single-page app without prerendering.
				adapter: adapter({ fallback: 'index.html', strict: true }),
				paths: { relative: false }
			})
		],
		server: {
			proxy: {
				[serverPaths]: { target: GO_ORIGIN }
			}
		}
	};
});
