import { redirect } from '@sveltejs/kit';
import { session } from '#lib/session.svelte.js';
import type { LayoutLoad } from './$types';

export const load: LayoutLoad = async ({ fetch }) => {
	const account = await session.load(fetch);
	if (!account) {
		redirect(307, '/auth');
	}
	if (account.onboarding_required) {
		redirect(307, '/auth/onboarding');
	}
};
