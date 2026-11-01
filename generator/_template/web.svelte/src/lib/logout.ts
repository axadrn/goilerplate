// Ends the session of this browser and returns to the sign-in page.
import { api, ApiError } from './api.js';
import { notifyError } from './notify.js';
import { session } from './session.svelte.js';

export async function logout(): Promise<void> {
	try {
		await api.post('/api/v1/auth/logout');
	} catch (cause) {
		// An expired session is already signed out.
		if (!(cause instanceof ApiError) || cause.status !== 401) {
			notifyError('Sign out failed', 'Please try again.');
			return;
		}
	}
	leaveApp();
}

// leaveApp opens the sign-in page with a full page load once the session is
// gone. That drops every bit of in-memory state, and a later Back starts a
// fresh load that redirects to the sign-in page again.
export function leaveApp(): void {
	session.signedOut();
	window.location.assign('/auth');
}
