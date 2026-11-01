import { error, redirect } from '@sveltejs/kit';
import { session } from '#lib/session.svelte.js';
import type { LayoutLoad } from './$types';

// The admin pages are for operators. Everyone else gets the 404 page, like
// the HTML admin.
export const load: LayoutLoad = async ({ fetch }) => {
	const account = await session.load(fetch);
	if (!account) {
		redirect(307, '/auth');
	}
	if (account.onboarding_required) {
		redirect(307, '/auth/onboarding');
	}
	if (!account.is_admin) {
		error(404, 'Not found');
	}
};
