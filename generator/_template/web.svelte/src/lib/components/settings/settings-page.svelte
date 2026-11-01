<script lang="ts">
	import ShieldIcon from '@lucide/svelte/icons/shield';
	import UserIcon from '@lucide/svelte/icons/user';
	import { onMount } from 'svelte';
	import { api } from '#lib/api.js';
	import * as Tabs from '#lib/components/ui/tabs/index.js';
	import { config } from '#lib/config.js';
	import { notifyInfo } from '#lib/notify.js';
	import { session } from '#lib/session.svelte.js';
	import type { AccountSession, APIToken, List } from '#lib/types.js';
	import ApiTokensCard from './api-tokens-card.svelte';
	import AvatarCard from './avatar-card.svelte';
	import DangerZoneCard from './danger-zone-card.svelte';
	import EmailCard from './email-card.svelte';
	import NameCard from './name-card.svelte';
	import PasswordCard from './password-card.svelte';
	import SessionsCard from './sessions-card.svelte';

	let {
		sessions: initialSessions,
		tokens,
		tab,
		passwordRemoved,
		workspaceId = ''
	}: {
		sessions: AccountSession[];
		tokens: APIToken[];
		tab: string;
		passwordRemoved: boolean;
		workspaceId?: string;
	} = $props();

	let sessions = $derived(initialSessions);
	let active = $derived(tab);

	onMount(() => {
		if (passwordRemoved) {
			notifyInfo('Password Removed', "For security, your password was removed. Set a new one below if you'd like.", 8000);
		}
	});

	async function reloadSessions() {
		sessions = (await api.get<List<AccountSession>>('/api/v1/account/sessions')).items;
	}
</script>

{#if session.account}
	{@const account = session.account}
	<div class="container max-w-4xl px-6 py-8">
		<div class="mb-8">
			<h1 class="text-3xl font-bold">Settings</h1>
			<p class="mt-2 text-muted-foreground">Manage your account settings</p>
		</div>
		<Tabs.Root bind:value={active}>
			<Tabs.List>
				<Tabs.Trigger value="profile">
					<UserIcon class="size-4" />
					Profile
				</Tabs.Trigger>
				<Tabs.Trigger value="security">
					<ShieldIcon class="size-4" />
					Security
				</Tabs.Trigger>
			</Tabs.List>
			<Tabs.Content value="profile">
				<div class="mt-6 space-y-6">
					<NameCard user={account.user} />
					{#if config.storage}
						<AvatarCard {account} />
					{/if}
				</div>
			</Tabs.Content>
			<Tabs.Content value="security">
				<div class="mt-6 space-y-6">
					<EmailCard user={account.user} />
					<PasswordCard user={account.user} onsessions={() => reloadSessions().catch(() => {})} />
					<SessionsCard {sessions} onreload={reloadSessions} />
					<ApiTokensCard {tokens} workspaces={account.workspaces} {workspaceId} />
					<DangerZoneCard user={account.user} />
				</div>
			</Tabs.Content>
		</Tabs.Root>
	</div>
{/if}
