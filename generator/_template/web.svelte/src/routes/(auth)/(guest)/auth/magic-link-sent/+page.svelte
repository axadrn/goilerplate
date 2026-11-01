<script lang="ts">
	import CheckIcon from '@lucide/svelte/icons/check';
	import { api } from '#lib/api.js';
	import AuthCard from '#lib/components/auth/auth-card.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { notifyFailure } from '#lib/notify.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	let sent = $state(false);
	let pending = $state(false);

	async function resend(event: SubmitEvent) {
		event.preventDefault();
		if (pending) {
			return;
		}
		pending = true;
		try {
			await api.post('/api/v1/auth/magic-link', { body: { email: data.email } });
			sent = true;
		} catch (cause) {
			notifyFailure(cause);
		} finally {
			pending = false;
		}
	}
</script>

<AuthCard title="Check your email" description="We sent a sign-in link to {data.email}">
	<Field.Group>
		<Field.Field>
			<div id="magic-link-resend" class="space-y-2">
				<form id="magic-link-resend-form" onsubmit={resend} aria-busy={pending || undefined}>
					<Button variant="outline" class="w-full" type="submit">
						{sent ? 'Resend again' : 'Resend email'}
					</Button>
				</form>
				{#if sent}
					<p role="status" class="flex items-center justify-center gap-1.5 text-center text-sm text-muted-foreground">
						<CheckIcon class="size-4" />
						Email sent. A new sign-in link is on its way.
					</p>
				{/if}
			</div>
			<Field.Description class="text-center">
				<a href="/auth">Use another email</a>
			</Field.Description>
		</Field.Field>
	</Field.Group>
</AuthCard>
