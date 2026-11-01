// App paths that depend on the selected modules.
import { config } from './config.js';
import type { Account } from './types.js';

// settingsPath answers the account settings page: /app/settings, or the
// settings of the first workspace with workspaces.
export function settingsPath(account: Account | null): string {
	if (!config.workspaces) {
		return '/app/settings';
	}
	const first = account?.workspaces?.[0];
	return first ? `/app/${first.workspace.slug}/settings` : '/app';
}
