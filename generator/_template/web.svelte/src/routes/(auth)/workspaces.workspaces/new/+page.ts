import { redirect } from '@sveltejs/kit';
import { session } from '#lib/session.svelte.js';
import type { PageLoad } from './$types';

// A signed-in, onboarded account creates another workspace here.
export const load: PageLoad = async ({ fetch }) => {
	const account = await session.current(fetch);
	if (!account) {
		redirect(307, '/auth');
	}
	if (account.onboarding_required) {
		redirect(307, '/auth/onboarding');
	}
	return { title: 'New workspace' };
};
