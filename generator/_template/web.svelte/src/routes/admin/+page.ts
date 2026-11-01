import { error, redirect } from '@sveltejs/kit';
import { api, ApiError } from '#lib/api.js';
import type { AdminDashboard } from '#lib/types.js';
import type { PageLoad } from './$types';

// ?q= filters users by name or email and ?page= pages through them, like the
// HTML admin.
export const load: PageLoad = async ({ fetch, parent, url }) => {
	await parent();
	const page = Number(url.searchParams.get('page'));
	let dashboard: AdminDashboard;
	try {
		dashboard = await api.get<AdminDashboard>('/api/v1/admin/users', {
			fetch,
			query: {
				query: url.searchParams.get('q') || undefined,
				page: Number.isInteger(page) && page > 0 ? page : undefined
			}
		});
	} catch (cause) {
		if (cause instanceof ApiError && cause.status === 401) {
			redirect(307, '/auth');
		}
		if (cause instanceof ApiError && cause.status === 403) {
			error(404, 'Not found');
		}
		error(500, 'The admin dashboard could not be loaded. Please try again.');
	}
	return { title: 'Admin', dashboard };
};
