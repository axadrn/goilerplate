import { redirect } from '@sveltejs/kit';
import { session } from '#lib/session.svelte.js';
import type { PageLoad } from './$types';

// Only a signed-in account without a name completes its profile.
export const load: PageLoad = async ({ fetch }) => {
	const account = await session.current(fetch);
	if (!account) {
		redirect(307, '/auth');
	}
	if (!account.onboarding_required) {
		redirect(307, '/app');
	}
	return { title: 'Complete Your Profile', email: account.user.email };
};
