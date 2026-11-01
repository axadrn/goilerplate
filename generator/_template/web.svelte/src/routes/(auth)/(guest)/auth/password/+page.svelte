<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount, tick } from 'svelte';
	import { api, ApiError } from '#lib/api.js';
	import AuthCard from '#lib/components/auth/auth-card.svelte';
	import OAuthButtons from '#lib/components/auth/oauth-buttons.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { config } from '#lib/config.js';
	import { notifyFailure } from '#lib/notify.js';
	import { session } from '#lib/session.svelte.js';
	import type { CookieSession } from '#lib/types.js';

	let email = $state('');
	let password = $state('');
	let message = $state('');
	let pending = $state(false);
	let emailInput = $state<HTMLInputElement | null>(null);

	onMount(() => emailInput?.focus());

	async function fail(text: string) {
		message = text;
		pending = false;
		await tick();
		emailInput?.focus();
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		const value = email.trim();
		if (!value || !password) {
			await fail('Email and password are required');
			return;
		}
		pending = true;
		let answer: CookieSession;
		try {
			answer = await api.post<CookieSession>('/api/v1/auth/password', {
				body: { email: value, password },
				query: { session: 'cookie' }
			});
		} catch (cause) {
			if (cause instanceof ApiError && cause.status === 422) {
				await fail(cause.fields.email ? 'Please provide a valid email address' : 'Email and password are required');
				return;
			}
			if (cause instanceof ApiError && cause.status === 401) {
				await fail('Invalid email or password');
				return;
			}
			if (cause instanceof ApiError && cause.status >= 500) {
				await fail('An error occurred. Please try again.');
				return;
			}
			pending = false;
			notifyFailure(cause);
			return;
		}
		session.signedIn();
		await goto(answer.user.name ? '/app' : '/auth/onboarding');
	}
</script>

<AuthCard title="Sign in" description="Enter your email and password to continue">
	<Field.Group>
		<form
			id="password-form"
			method="POST"
			onsubmit={submit}
			aria-busy={pending || undefined}
			inert={pending}
		>
			<Field.Group>
				<Field.Field>
					<Field.Label for="email">Email</Field.Label>
					<Input
						bind:ref={emailInput}
						bind:value={email}
						id="email"
						name="email"
						type="email"
						placeholder="name@example.com"
						autocomplete="email"
						aria-invalid={message ? 'true' : 'false'}
						aria-describedby={message ? 'password-form-error' : undefined}
					/>
					{#if message}
						<Field.Error id="password-form-error">{message}</Field.Error>
					{/if}
				</Field.Field>
				<Field.Field>
					<div class="flex items-center">
						<Field.Label for="password">Password</Field.Label>
						<a href="/auth/forgot-password" class="ml-auto text-sm underline-offset-4 hover:underline">
							Forgot password?
						</a>
					</div>
					<Input
						bind:value={password}
						id="password"
						name="password"
						type="password"
						placeholder="••••••••"
						autocomplete="current-password"
					/>
				</Field.Field>
				<Button type="submit" class="w-full">Sign in</Button>
			</Field.Group>
		</form>
		{#if config.google || config.github}
			<Field.Separator class="*:data-[slot=field-separator-content]:bg-card">Or continue with</Field.Separator>
			<OAuthButtons />
		{/if}
		<Field.Separator class="*:data-[slot=field-separator-content]:bg-card">Or</Field.Separator>
		<Button href="/auth" variant="outline" class="w-full">Continue with magic link</Button>
	</Field.Group>
</AuthCard>
