<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import CircleAlertIcon from '@lucide/svelte/icons/circle-alert';
	import CircleCheckIcon from '@lucide/svelte/icons/circle-check';
	import CircleXIcon from '@lucide/svelte/icons/circle-x';
	import { onMount } from 'svelte';
	import { api, ApiError } from '#lib/api.js';
	import AuthCard from '#lib/components/auth/auth-card.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import { config } from '#lib/config.js';
	import { settingsPath } from '#lib/paths.js';
	import { session } from '#lib/session.svelte.js';

	let status = $state<'pending' | 'verified' | 'failed'>('pending');
	let error = $state('');
	let settings = $state('/app');

	const title = $derived(
		status === 'verified' ? 'Email Changed' : status === 'failed' ? 'Verification Failed' : 'Verify Email'
	);

	// The link moves the account to the new address once, signs out every
	// other session and signs this browser in again.
	onMount(() => {
		void verify();
	});

	// Like the HTML page, a verified change moves on to settings after three
	// seconds.
	$effect(() => {
		if (status !== 'verified') {
			return;
		}
		const timer = setTimeout(() => goto(settings), 3000);
		return () => clearTimeout(timer);
	});

	async function verify(): Promise<void> {
		try {
			await api.post('/api/v1/auth/email-change/verify', {
				body: { token: page.params.token ?? '' },
				query: { session: 'cookie' }
			});
		} catch (cause) {
			error =
				cause instanceof ApiError && cause.status === 400
					? 'Invalid or expired verification link'
					: 'An error occurred. Please try again.';
			status = 'failed';
			return;
		}
		const account = await session.refresh().catch(() => null);
		settings = settingsPath(account);
		status = 'verified';
	}
</script>

<svelte:head>
	<title>{title} - {config.name}</title>
</svelte:head>

{#if status === 'verified'}
	<AuthCard title="Email verified" description="Your email address has been updated">
		<Empty.Root class="border-0 p-0">
			<Empty.Header>
				<Empty.Media variant="icon" class="bg-emerald-500/10 text-emerald-700 dark:text-emerald-300">
					<CircleCheckIcon />
				</Empty.Media>
				<Empty.Description>Redirecting you to settings.</Empty.Description>
			</Empty.Header>
			<Empty.Content>
				<Button href={settings} class="w-full">Go to settings</Button>
			</Empty.Content>
		</Empty.Root>
	</AuthCard>
{:else if status === 'failed'}
	<AuthCard title="Verification failed" description="We couldn't verify your email">
		<Empty.Root class="border-0 p-0">
			<Empty.Header>
				<Empty.Media variant="icon" class="bg-destructive/10 text-destructive">
					<CircleXIcon />
				</Empty.Media>
				<Empty.Description>The verification link may have expired or already been used.</Empty.Description>
			</Empty.Header>
			<Empty.Content class="max-w-none">
				<Alert.Root variant="destructive">
					<CircleAlertIcon />
					<Alert.Description>{error}</Alert.Description>
				</Alert.Root>
				<Button href="/auth" class="w-full">Try again</Button>
			</Empty.Content>
		</Empty.Root>
	</AuthCard>
{:else}
	<p role="status" class="sr-only">Verifying your email</p>
{/if}
