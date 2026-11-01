import type { PageLoad } from './$types';

// The Go server sends failed OAuth sign-ins here with ?error=oauth_failed.
export const load: PageLoad = ({ url }) => ({
	title: 'Welcome',
	error: url.searchParams.get('error') === 'oauth_failed' ? 'OAuth authentication failed. Please try again.' : ''
});
