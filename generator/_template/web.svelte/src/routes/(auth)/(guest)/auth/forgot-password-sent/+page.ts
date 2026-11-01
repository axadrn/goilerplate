import { redirect } from '@sveltejs/kit';
import { authEmail } from '#lib/auth-email.js';

export function load() {
	const email = authEmail.get();
	if (!email) {
		redirect(307, '/auth/forgot-password');
	}
	return { title: 'Check Your Email', email };
}
