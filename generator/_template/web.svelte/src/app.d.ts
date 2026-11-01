// See https://svelte.dev/docs/kit/types#app.d.ts
import type { WorkspaceDetail } from '#lib/types.js';

declare global {
	namespace App {
		interface Error {
			message: string;
			// The heading of the error page, like the HTML error pages.
			title?: string;
		}
		interface PageData {
			// The page heading in the app header and the document title.
			title?: string;
			// The workspace of /app/[workspaceSlug] pages, with workspaces.
			workspace?: WorkspaceDetail;
		}
	}
}

export {};
