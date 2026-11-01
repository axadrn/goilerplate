<script lang="ts">
	import LogOutIcon from '@lucide/svelte/icons/log-out';
	import MonitorIcon from '@lucide/svelte/icons/monitor';
	import { tick } from 'svelte';
	import { api, ApiError } from '#lib/api.js';
	import * as AlertDialog from '#lib/components/ui/alert-dialog/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { formatDayTime } from '#lib/format.js';
	import { logout } from '#lib/logout.js';
	import { notifyError, notifyFailure, notifySuccess } from '#lib/notify.js';
	import type { AccountSession } from '#lib/types.js';

	let { sessions, onreload }: { sessions: AccountSession[]; onreload: () => Promise<void> } = $props();

	let busy = $state<string | null>(null);
	let othersOpen = $state(false);
	let revokingOthers = $state(false);
	let othersButton = $state<HTMLElement | null>(null);

	async function revoke(item: AccountSession) {
		if (busy) {
			return;
		}
		if (item.current) {
			busy = item.id;
			await logout();
			busy = null;
			return;
		}
		busy = item.id;
		try {
			await api.delete(`/api/v1/account/sessions/${encodeURIComponent(item.id)}`);
			notifySuccess('Session revoked', 'The session has been signed out.');
		} catch (cause) {
			if (cause instanceof ApiError && cause.status === 404) {
				notifyError('Session not found', 'That session has already ended.');
			} else {
				notifyFailure(cause, 'Session could not be revoked', 'Please try again.');
			}
		}
		await reload();
		busy = null;
		// The revoked row and its button are gone.
		await tick();
		if (!document.activeElement || document.activeElement === document.body) {
			othersButton?.focus();
		}
	}

	async function revokeOthers() {
		if (revokingOthers) {
			return;
		}
		revokingOthers = true;
		try {
			await api.delete('/api/v1/account/sessions');
			notifySuccess('Sessions revoked', 'Every other session has been signed out.');
		} catch (cause) {
			notifyFailure(cause, 'Sessions could not be revoked', 'Please try again.');
		}
		othersOpen = false;
		revokingOthers = false;
		await reload();
	}

	async function reload() {
		try {
			await onreload();
		} catch (cause) {
			notifyFailure(cause, 'Sessions could not be refreshed', 'Please try again.');
		}
	}
</script>

<Card.Root id="settings-sessions">
	<Card.Header>
		<Card.Title>Active Sessions</Card.Title>
		<Card.Description>Review the browsers and devices currently signed in to your account</Card.Description>
	</Card.Header>
	<Card.Content>
		<ul class="divide-y rounded-lg border" aria-label="Active sessions">
			{#each sessions as item (item.id)}
				<li class="flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between">
					<div class="flex min-w-0 items-start gap-3">
						<div class="mt-0.5 rounded-md bg-muted p-2">
							<MonitorIcon class="size-4" />
						</div>
						<div class="min-w-0 space-y-1">
							<div class="flex flex-wrap items-center gap-2">
								<p class="font-medium">{item.display_name}</p>
								{#if item.current}
									<Badge variant="secondary">Current</Badge>
								{/if}
							</div>
							<p class="text-xs text-muted-foreground">Last active {formatDayTime(item.last_seen_at)}</p>
						</div>
					</div>
					<Button
						variant="outline"
						class="w-full sm:w-auto"
						aria-label={item.current ? undefined : `Revoke ${item.display_name}`}
						aria-busy={busy === item.id || undefined}
						onclick={() => revoke(item)}
					>
						{item.current ? 'Sign Out' : 'Revoke'}
					</Button>
				</li>
			{/each}
		</ul>
		<div class="mt-4 flex justify-end">
			<Button bind:ref={othersButton} variant="outline" aria-haspopup="dialog" onclick={() => (othersOpen = true)}>
				<LogOutIcon class="mr-2 size-4" />
				Sign Out Everywhere Else
			</Button>
		</div>
	</Card.Content>
</Card.Root>

<AlertDialog.Root bind:open={othersOpen}>
	<AlertDialog.Content>
		<AlertDialog.Header>
			<AlertDialog.Title>Sign Out Everywhere Else</AlertDialog.Title>
			<AlertDialog.Description>Sign out of every other session? This browser stays signed in.</AlertDialog.Description>
		</AlertDialog.Header>
		<AlertDialog.Footer>
			<AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
			<AlertDialog.Action onclick={revokeOthers} aria-busy={revokingOthers || undefined}>Sign Out Others</AlertDialog.Action>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>
