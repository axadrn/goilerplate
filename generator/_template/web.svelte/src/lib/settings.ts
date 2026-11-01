// Loads the account settings page: the sessions and API tokens of the
// signed-in account, and the tab and notice the address asks for.
import { error } from '@sveltejs/kit';
import { api } from './api.js';
import type { AccountSession, APIToken, List } from './types.js';

export async function loadSettings(fetcher: typeof fetch, url: URL) {
	let sessions: List<AccountSession>;
	let tokens: List<APIToken>;
	try {
		[sessions, tokens] = await Promise.all([
			api.get<List<AccountSession>>('/api/v1/account/sessions', { fetch: fetcher }),
			api.get<List<APIToken>>('/api/v1/account/tokens', { fetch: fetcher })
		]);
	} catch {
		error(500, 'Your settings could not be loaded. Please try again.');
	}
	return {
		title: 'Settings',
		sessions: sessions.items,
		tokens: tokens.items,
		tab: url.searchParams.get('tab') === 'security' ? 'security' : 'profile',
		passwordRemoved: url.searchParams.get('password_removed') === '1'
	};
}
