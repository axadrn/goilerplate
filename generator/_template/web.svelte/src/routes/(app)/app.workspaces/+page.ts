import { redirect } from '@sveltejs/kit';
import { session } from '#lib/session.svelte.js';
import { firstWorkspacePath } from '#lib/workspace/workspace.js';
import type { PageLoad } from './$types';

// /app returns to an invitation this browser opened before signing in. The
// server keeps it in the pending_invitation cookie, like the HTML app, and
// clears it once the signed-in invitation page loads. Otherwise /app opens
// the first workspace, or asks for a new one.
export const load: PageLoad = async ({ parent }) => {
	await parent();
	const invitation = session.account?.pending_invitation;
	if (invitation) {
		redirect(307, `/workspaces/invitations/${encodeURIComponent(invitation)}`);
	}
	redirect(307, firstWorkspacePath(session.account));
};
