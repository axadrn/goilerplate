// The signed-in account and the public server settings. The app layout loads
// both; pages that change the account call refresh().
import { api, ApiError } from './api.js';
import type { Account, Meta } from './types.js';

class Session {
	account = $state<Account | null>(null);
	meta = $state<Meta | null>(null);
	// Whether account reflects the server since the last sign-in or sign-out.
	#known = false;

	// load answers the account, or null without a valid session. Load
	// functions pass their fetch.
	async load(fetcher?: typeof fetch): Promise<Account | null> {
		const [account, meta] = await Promise.all([
			this.refresh(fetcher),
			this.meta ?? api.get<Meta>('/api/v1/meta', { fetch: fetcher })
		]);
		this.meta = meta;
		return account;
	}

	async refresh(fetcher?: typeof fetch): Promise<Account | null> {
		try {
			this.account = await api.get<Account>('/api/v1/account', { fetch: fetcher });
		} catch (error) {
			if (!(error instanceof ApiError) || error.status !== 401) {
				throw error;
			}
			this.account = null;
		}
		this.#known = true;
		return this.account;
	}

	// current answers the account without asking the server again when the
	// answer is already known. Guest pages use it.
	current(fetcher?: typeof fetch): Promise<Account | null> {
		return this.#known ? Promise.resolve(this.account) : this.refresh(fetcher);
	}

	// signedIn marks the account as stale after a sign-in, so the next page
	// asks the server.
	signedIn(): void {
		this.#known = false;
	}

	// signedOut forgets the account after a sign-out.
	signedOut(): void {
		this.account = null;
		this.#known = true;
	}
}

export const session = new Session();
