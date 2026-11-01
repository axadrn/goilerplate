import { error } from '@sveltejs/kit';
import { api } from '#lib/api.js';
import type { ActivityPage } from '#lib/types.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent }) => {
	const { workspace } = await parent();
	if (!workspace.permissions.view_audit_log) {
		error(403, { title: 'Access denied', message: 'You do not have permission to view workspace activity.' });
	}
	let first: ActivityPage;
	try {
		first = await api.get<ActivityPage>(`/api/v1/workspaces/${encodeURIComponent(workspace.workspace.slug)}/activity`, {
			fetch
		});
	} catch {
		error(500, { title: 'Activity unavailable', message: 'Workspace activity could not be loaded. Please try again.' });
	}
	return { title: 'Activity', first };
};
