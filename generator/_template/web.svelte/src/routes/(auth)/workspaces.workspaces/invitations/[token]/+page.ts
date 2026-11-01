import { api, ApiError } from '#lib/api.js';
import { session } from '#lib/session.svelte.js';
import type { InvitationPreview } from '#lib/types.js';
import type { PageLoad } from './$types';

// Anyone with the link sees who invites to which workspace. For guests the
// preview answer sets the pending_invitation cookie that sign-in and
// onboarding return with.
export const load: PageLoad = async ({ fetch, params }) => {
	const account = await session.current(fetch);
	let preview: InvitationPreview | null = null;
	let error = '';
	try {
		preview = await api.get<InvitationPreview>(`/api/v1/invitations/${encodeURIComponent(params.token)}`, { fetch });
	} catch (cause) {
		error = invitationLoadError(cause);
	}
	return { title: 'Workspace invitation', account, preview, error };
};

function invitationLoadError(cause: unknown): string {
	if (cause instanceof ApiError && cause.status === 410) {
		return 'This invitation is no longer available.';
	}
	if (cause instanceof ApiError && cause.status === 404) {
		return 'This invitation is invalid or no longer available.';
	}
	return 'The invitation could not be loaded. Please try again.';
}
