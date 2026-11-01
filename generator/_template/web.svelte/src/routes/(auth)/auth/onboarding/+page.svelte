<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount, tick } from 'svelte';
	import { api, ApiError } from '#lib/api.js';
	import AuthCard from '#lib/components/auth/auth-card.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { logout } from '#lib/logout.js';
	import { notifyFailure } from '#lib/notify.js';
	import { session } from '#lib/session.svelte.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	let name = $state('');
	let message = $state('');
	let pending = $state(false);
	let signingOut = $state(false);
	let input = $state<HTMLInputElement | null>(null);

	onMount(() => input?.focus());

	async function fail(text: string) {
		message = text;
		pending = false;
		await tick();
		input?.focus();
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (!name.trim()) {
			await fail('Please enter your name');
			return;
		}
		pending = true;
		try {
			await api.post('/api/v1/auth/onboarding', { body: { name: name.trim() } });
		} catch (cause) {
			if (cause instanceof ApiError && cause.status === 429) {
				pending = false;
				notifyFailure(cause);
				return;
			}
			await fail('Please enter your name');
			return;
		}
		session.signedIn();
		await goto('/app');
	}

	async function signOut(event: SubmitEvent) {
		event.preventDefault();
		if (signingOut) {
			return;
		}
		signingOut = true;
		await logout();
		signingOut = false;
	}
</script>

<AuthCard title="Complete your profile" description="Choose the name people will see">
	<Field.Group>
		<form
			id="onboarding-form"
			method="POST"
			onsubmit={submit}
			aria-busy={pending || undefined}
			inert={pending}
		>
			<Field.Group>
				<Field.Field>
					<Field.Label for="name">Your name</Field.Label>
					<Input
						bind:ref={input}
						bind:value={name}
						id="name"
						name="name"
						type="text"
						placeholder="John Doe"
						autocomplete="name"
						aria-invalid={message ? 'true' : 'false'}
						aria-describedby={message ? 'name-description name-error' : 'name-description'}
					/>
					<Field.Description id="name-description">
						Continue as <strong>{data.email}</strong>.
					</Field.Description>
					{#if message}
						<Field.Error id="name-error">{message}</Field.Error>
					{/if}
				</Field.Field>
				<Button type="submit" class="w-full">Continue</Button>
			</Field.Group>
		</form>
		<Field.Separator>Not you?</Field.Separator>
		<form onsubmit={signOut} aria-busy={signingOut || undefined}>
			<Button type="submit" variant="outline" class="w-full">Sign out</Button>
		</form>
	</Field.Group>
</AuthCard>
