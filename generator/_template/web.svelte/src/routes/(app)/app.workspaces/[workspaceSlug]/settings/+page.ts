import { loadSettings } from '#lib/settings.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent, url }) => {
	await parent();
	return loadSettings(fetch, url);
};
