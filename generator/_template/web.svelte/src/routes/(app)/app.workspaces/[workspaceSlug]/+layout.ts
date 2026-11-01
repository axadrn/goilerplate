import { error } from '@sveltejs/kit';
import { api, ApiError } from '#lib/api.js';
import type { WorkspaceDetail } from '#lib/types.js';
import type { LayoutLoad } from './$types';

export const load: LayoutLoad = async ({ fetch, params, parent }) => {
	await parent();
	try {
		const path = `/api/v1/workspaces/${encodeURIComponent(params.workspaceSlug)}`;
		const workspace = await api.get<WorkspaceDetail>(path, { fetch });
		return { workspace };
	} catch (cause) {
		if (cause instanceof ApiError && (cause.status === 403 || cause.status === 404)) {
			error(404, 'Not found');
		}
		throw cause;
	}
};
