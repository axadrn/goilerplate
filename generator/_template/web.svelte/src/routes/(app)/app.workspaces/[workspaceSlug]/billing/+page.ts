import { loadBilling } from '#lib/billing/billing.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, params, parent }) => {
	await parent();
	return { title: 'Billing', billing: await loadBilling(fetch, params.workspaceSlug) };
};
