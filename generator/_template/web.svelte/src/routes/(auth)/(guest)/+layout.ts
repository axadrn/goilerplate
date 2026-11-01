import { redirect } from '@sveltejs/kit';
import { session } from '#lib/session.svelte.js';
import type { LayoutLoad } from './$types';

// Signed-in visitors skip the sign-in pages.
export const load: LayoutLoad = async ({ fetch }) => {
	const account = await session.current(fetch);
	if (account) {
		redirect(307, '/app');
	}
};
