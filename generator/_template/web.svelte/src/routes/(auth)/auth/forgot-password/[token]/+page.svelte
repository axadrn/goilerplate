<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import { api, ApiError } from '#lib/api.js';
	import SignIn from '#lib/components/auth/sign-in.svelte';
	import { settingsPath } from '#lib/paths.js';
	import { session } from '#lib/session.svelte.js';

	let error = $state('');

	// The reset link removes the password and signs in once, so the page sends
	// it once. Settings then says the password is gone.
	onMount(async () => {
		try {
			await api.post('/api/v1/auth/forgot-password/verify', {
				body: { token: page.params.token ?? '' },
				query: { session: 'cookie' }
			});
		} catch (cause) {
			error =
				cause instanceof ApiError && cause.status === 400
					? 'Invalid or expired link. Please try again.'
					: 'An error occurred. Please try again.';
			return;
		}
		const account = await session.refresh().catch(() => null);
		await goto(`${settingsPath(account)}?tab=security&password_removed=1`, { replace: true });
	});
</script>

{#if error}
	<SignIn {error} />
{:else}
	<p role="status" class="sr-only">Signing you in</p>
{/if}
