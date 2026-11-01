<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import { api, ApiError } from '#lib/api.js';
	import SignIn from '#lib/components/auth/sign-in.svelte';
	import { session } from '#lib/session.svelte.js';
	import type { CookieSession } from '#lib/types.js';

	let error = $state('');

	// A sign-in link works once, so the page sends it once.
	onMount(async () => {
		let answer: CookieSession;
		try {
			answer = await api.post<CookieSession>('/api/v1/auth/magic-link/verify', {
				body: { token: page.params.token ?? '' },
				query: { session: 'cookie' }
			});
		} catch (cause) {
			error =
				cause instanceof ApiError && cause.status === 400
					? 'Invalid or expired magic link. Please try again.'
					: 'An error occurred. Please try again.';
			return;
		}
		session.signedIn();
		await goto(answer.user.name ? '/app' : '/auth/onboarding', { replace: true });
	});
</script>

{#if error}
	<SignIn {error} />
{:else}
	<p role="status" class="sr-only">Signing you in</p>
{/if}
