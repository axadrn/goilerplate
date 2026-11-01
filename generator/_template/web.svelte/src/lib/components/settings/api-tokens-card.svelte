<script lang="ts">
	import CopyIcon from '@lucide/svelte/icons/copy';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import { tick } from 'svelte';
	import { api, ApiError } from '#lib/api.js';
	import * as AlertDialog from '#lib/components/ui/alert-dialog/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Dialog from '#lib/components/ui/dialog/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import { config } from '#lib/config.js';
	import { formatDay } from '#lib/format.js';
	import { notifyError, notifyFailure, notifyInfo, notifySuccess } from '#lib/notify.js';
	import type { APIToken, CreatedAPIToken, List, Membership } from '#lib/types.js';

	// workspaceId is the workspace a new token is bound to by default, with
	// workspaces.
	let {
		tokens: initialTokens,
		workspaces = [],
		workspaceId = ''
	}: { tokens: APIToken[]; workspaces?: Membership[]; workspaceId?: string } = $props();

	type FormField = 'name' | 'access' | 'expiry' | 'workspace_id' | 'form';

	const accessItems = [
		{ value: 'read', label: 'Read only' },
		{ value: 'write', label: 'Read and write' }
	];
	const expiryItems = [
		{ value: '7', label: '7 days' },
		{ value: '30', label: '30 days' },
		{ value: '90', label: '90 days' },
		{ value: '365', label: '1 year' },
		{ value: 'never', label: 'No expiration' }
	];
	const workspaceItems = $derived(workspaces.map((m) => ({ value: m.workspace.id, label: m.workspace.name })));
	const workspaceNames = $derived(new Map(workspaces.map((m) => [m.workspace.id, m.workspace.name])));

	let tokens = $derived(initialTokens);
	let createOpen = $state(false);
	let name = $state('');
	let access = $state('read');
	let expiry = $state('90');
	let workspace = $derived(workspaceId);
	let errors = $state<Partial<Record<FormField, string>>>({});
	let creating = $state(false);

	let raw = $state('');
	let revealOpen = $state(false);
	let copied = $state(false);
	let copyStatus = $state('');
	let tokenInput = $state<HTMLInputElement | null>(null);

	let revokeTarget = $state<APIToken | null>(null);
	let revokeOpen = $state(false);
	let revoking = $state(false);
	// Whether the revoked row is gone, so focus cannot go back to its button.
	let revoked = $state(false);

	let createButton = $state<HTMLElement | null>(null);
	let fieldRefs = $state<Record<Exclude<FormField, 'form'>, HTMLElement | null>>({
		name: null,
		access: null,
		expiry: null,
		workspace_id: null
	});

	const invalid = (field: FormField) => (errors[field] ? 'true' : 'false');

	function resetForm() {
		name = '';
		access = 'read';
		expiry = '90';
		workspace = workspaceId;
		errors = {};
	}

	async function reload(): Promise<boolean> {
		try {
			tokens = (await api.get<List<APIToken>>('/api/v1/account/tokens')).items;
			return true;
		} catch {
			return false;
		}
	}

	async function showErrors(next: Partial<Record<FormField, string>>) {
		errors = next;
		await tick();
		const first = (['name', 'access', 'expiry', 'workspace_id'] as const).find((field) => next[field]);
		fieldRefs[first ?? 'name']?.focus();
	}

	async function create(event: SubmitEvent) {
		event.preventDefault();
		if (creating) {
			return;
		}
		creating = true;
		let created: CreatedAPIToken;
		try {
			created = await api.post<CreatedAPIToken>('/api/v1/account/tokens', {
				body: {
					name,
					scopes: access === 'write' ? ['read', 'write'] : ['read'],
					expires_in_days: expiry === 'never' ? null : Number(expiry),
					...(config.workspaces ? { workspace_id: workspace } : {})
				}
			});
		} catch (cause) {
			creating = false;
			if (cause instanceof ApiError && cause.status === 422) {
				const next: Partial<Record<FormField, string>> = {};
				if (cause.fields.name) next.name = 'Enter a name with at most 64 characters.';
				if (cause.fields.scopes) next.access = 'Choose read only, or read and write.';
				if (cause.fields.expires_in_days) next.expiry = 'Choose an expiration.';
				if (cause.fields.workspace_id) next.workspace_id = 'Choose one of your workspaces.';
				await showErrors(next);
			} else if (cause instanceof ApiError && cause.code === 'token_limit') {
				await showErrors({ form: 'You have reached the limit of 50 API tokens. Revoke one first.' });
			} else {
				notifyFailure(cause, 'Token could not be created', 'Please try again.');
			}
			return;
		}
		creating = false;
		createOpen = false;
		resetForm();
		raw = created.token;
		copied = false;
		copyStatus = '';
		revealOpen = true;
		if (!(await reload())) {
			notifyInfo('Token created', 'Reload the page to see it in the list.');
		}
	}

	async function copy() {
		try {
			await navigator.clipboard.writeText(raw);
			copied = true;
			copyStatus = 'Token copied to the clipboard';
		} catch {
			tokenInput?.select();
			copyStatus = 'Copy failed. Select the token and copy it manually';
		}
	}

	function askRevoke(token: APIToken) {
		revokeTarget = token;
		revoked = false;
		revokeOpen = true;
	}

	async function revoke() {
		const target = revokeTarget;
		if (!target || revoking) {
			return;
		}
		revoking = true;
		try {
			await api.delete(`/api/v1/account/tokens/${encodeURIComponent(target.id)}`);
			revoked = true;
			notifySuccess('Token revoked', 'Requests with it no longer work.');
		} catch (cause) {
			if (cause instanceof ApiError && cause.status === 404) {
				revoked = true;
				notifyError('Token not found', 'That token was already revoked.');
			} else {
				notifyFailure(cause, 'Token could not be revoked', 'Please try again.');
				revoking = false;
				revokeOpen = false;
				return;
			}
		}
		await reload();
		revoking = false;
		revokeOpen = false;
	}

	function accessLabel(token: APIToken): string {
		return token.scopes.includes('write') ? 'Read and write' : 'Read only';
	}

	function lastUsed(token: APIToken): string {
		return token.last_used_at ? `Last used ${formatDay(token.last_used_at)}` : 'Never used';
	}

	function expires(token: APIToken): string {
		if (!token.expires_at) {
			return 'No expiration';
		}
		return `${token.expired ? 'Expired' : 'Expires'} ${formatDay(token.expires_at)}`;
	}

	function workspaceName(token: APIToken): string {
		if (!config.workspaces || !token.workspace_id) {
			return '';
		}
		return workspaceNames.get(token.workspace_id) ?? 'A workspace you left';
	}

	// Focus the create button when a dialog closes without a better target.
	function focusCreate(event: Event) {
		event.preventDefault();
		createButton?.focus();
	}
</script>

<Card.Root id="settings-api-tokens">
	<Card.Header>
		<Card.Title>API Tokens</Card.Title>
		<Card.Description>
			Personal tokens let scripts and AI agents use the JSON API as you. Treat them like passwords.
		</Card.Description>
		<Card.Action>
			<Dialog.Root bind:open={createOpen}>
				<Dialog.Trigger>
					{#snippet child({ props })}
						<Button bind:ref={createButton} size="sm" {...props}>
							<PlusIcon class="size-4" />
							Create Token
						</Button>
					{/snippet}
				</Dialog.Trigger>
				<Dialog.Content class="sm:max-w-md">
					<Dialog.Header>
						<Dialog.Title>Create API Token</Dialog.Title>
						<Dialog.Description>The token is shown once after you create it.</Dialog.Description>
					</Dialog.Header>
					<form id="api-token-form" class="space-y-4" onsubmit={create} aria-busy={creating || undefined}>
						<Field.Field>
							<Field.Label for="api-token-name">Name</Field.Label>
							<Input
								bind:ref={fieldRefs.name}
								bind:value={name}
								id="api-token-name"
								name="name"
								type="text"
								placeholder="Claude Desktop"
								required
								maxlength={64}
								autocomplete="off"
								aria-invalid={invalid('name')}
								aria-describedby={errors.name ? 'api-token-name-error' : undefined}
							/>
							{#if errors.name}
								<Field.Error id="api-token-name-error">{errors.name}</Field.Error>
							{/if}
						</Field.Field>
						<Field.Field>
							<Field.Label for="api-token-access">Access</Field.Label>
							<Select.Root type="single" name="access" bind:value={access} items={accessItems}>
								<Select.Trigger
									bind:ref={fieldRefs.access}
									id="api-token-access"
									class="w-full"
									aria-invalid={invalid('access')}
									aria-describedby="api-token-access-description"
								>
									<Select.Value />
								</Select.Trigger>
								<Select.Content align="start">
									{#each accessItems as item (item.value)}
										<Select.Item value={item.value} label={item.label} />
									{/each}
								</Select.Content>
							</Select.Root>
							<Field.Description id="api-token-access-description">
								Read only tokens can look but not change anything.
							</Field.Description>
							{#if errors.access}
								<Field.Error>{errors.access}</Field.Error>
							{/if}
						</Field.Field>
						<Field.Field>
							<Field.Label for="api-token-expiry">Expiration</Field.Label>
							<Select.Root type="single" name="expiry" bind:value={expiry} items={expiryItems}>
								<Select.Trigger
									bind:ref={fieldRefs.expiry}
									id="api-token-expiry"
									class="w-full"
									aria-invalid={invalid('expiry')}
								>
									<Select.Value />
								</Select.Trigger>
								<Select.Content align="start">
									{#each expiryItems as item (item.value)}
										<Select.Item value={item.value} label={item.label} />
									{/each}
								</Select.Content>
							</Select.Root>
							{#if errors.expiry}
								<Field.Error>{errors.expiry}</Field.Error>
							{/if}
						</Field.Field>
						{#if config.workspaces && workspaceItems.length > 0}
							<Field.Field>
								<Field.Label for="api-token-workspace">Workspace</Field.Label>
								<Select.Root type="single" name="workspace_id" bind:value={workspace} items={workspaceItems}>
									<Select.Trigger
										bind:ref={fieldRefs.workspace_id}
										id="api-token-workspace"
										class="w-full"
										aria-invalid={invalid('workspace_id')}
										aria-describedby="api-token-workspace-description"
									>
										<Select.Value />
									</Select.Trigger>
									<Select.Content align="start">
										{#each workspaceItems as item (item.value)}
											<Select.Item value={item.value} label={item.label} />
										{/each}
									</Select.Content>
								</Select.Root>
								<Field.Description id="api-token-workspace-description">
									The token can only reach this workspace.
								</Field.Description>
								{#if errors.workspace_id}
									<Field.Error>{errors.workspace_id}</Field.Error>
								{/if}
							</Field.Field>
						{/if}
						{#if errors.form}
							<Field.Error>{errors.form}</Field.Error>
						{/if}
						<Dialog.Footer>
							<Dialog.Close>
								{#snippet child({ props })}
									<Button variant="outline" {...props}>Cancel</Button>
								{/snippet}
							</Dialog.Close>
							<Button type="submit">Create Token</Button>
						</Dialog.Footer>
					</form>
				</Dialog.Content>
			</Dialog.Root>
		</Card.Action>
	</Card.Header>
	<Card.Content>
		<div id="api-token-list">
			{#if tokens.length === 0}
				<p class="rounded-lg border border-dashed p-6 text-center text-sm text-muted-foreground">
					No API tokens yet. Create one to connect a script or an AI agent.
				</p>
			{:else}
				<div class="overflow-hidden rounded-lg border">
					<Table.Root>
						<Table.Header>
							<Table.Row>
								<Table.Head class="px-4">Token</Table.Head>
								<Table.Head class="hidden px-4 sm:table-cell">Access</Table.Head>
								<Table.Head class="hidden px-4 md:table-cell">Usage</Table.Head>
								<Table.Head class="px-4 text-right"><span class="sr-only">Actions</span></Table.Head>
							</Table.Row>
						</Table.Header>
						<Table.Body>
							{#each tokens as token (token.id)}
								<Table.Row data-api-token-row={token.id}>
									<Table.Cell class="px-4 py-3 align-top whitespace-normal">
										<div class="flex flex-wrap items-center gap-2">
											<span class="font-medium break-all">{token.name}</span>
											{#if token.expired}
												<Badge variant="destructive">Expired</Badge>
											{/if}
										</div>
										<p class="mt-1 font-mono text-xs text-muted-foreground">pat_{token.prefix}…</p>
										{#if workspaceName(token)}
											<p class="mt-1 text-xs text-muted-foreground">{workspaceName(token)}</p>
										{/if}
										<p class="mt-1 text-xs text-muted-foreground sm:hidden">{accessLabel(token)}</p>
										<p class="mt-1 text-xs text-muted-foreground md:hidden">{lastUsed(token)} · {expires(token)}</p>
									</Table.Cell>
									<Table.Cell class="hidden px-4 py-3 align-top sm:table-cell">
										<Badge variant={token.scopes.includes('write') ? 'default' : 'secondary'}>{accessLabel(token)}</Badge>
									</Table.Cell>
									<Table.Cell class="hidden px-4 py-3 align-top text-xs text-muted-foreground md:table-cell">
										<p>{lastUsed(token)}</p>
										<p class="mt-1">{expires(token)}</p>
									</Table.Cell>
									<Table.Cell class="px-4 py-3 text-right align-top">
										<Button
											variant="outline"
											size="sm"
											aria-label="Revoke {token.name}"
											aria-haspopup="dialog"
											onclick={() => askRevoke(token)}
										>
											Revoke
										</Button>
									</Table.Cell>
								</Table.Row>
							{/each}
						</Table.Body>
					</Table.Root>
				</div>
			{/if}
		</div>
	</Card.Content>
</Card.Root>

<!-- The raw token exists only while this dialog is open. -->
<Dialog.Root bind:open={revealOpen} onOpenChange={(open) => open || (raw = '')}>
	<Dialog.Content class="sm:max-w-md" interactOutsideBehavior="ignore" onCloseAutoFocus={focusCreate}>
		<Dialog.Header>
			<Dialog.Title>Copy Your New Token</Dialog.Title>
			<Dialog.Description>
				This is the only time the token is shown. Store it somewhere safe, for example in your password manager.
			</Dialog.Description>
		</Dialog.Header>
		{#if raw}
			<Field.Field>
				<Field.Label for="api-token-value">Token</Field.Label>
				<div class="flex gap-2">
					<Input
						bind:ref={tokenInput}
						id="api-token-value"
						type="text"
						value={raw}
						readonly
						class="font-mono text-xs"
						autocomplete="off"
						spellcheck="false"
					/>
					<Button variant="outline" onclick={copy}>
						<CopyIcon class="size-4" />
						<span>{copied ? 'Copied' : 'Copy'}</span>
					</Button>
				</div>
				<p class="sr-only" role="status" aria-live="polite">{copyStatus}</p>
			</Field.Field>
		{/if}
		<Dialog.Footer>
			<Dialog.Close>
				{#snippet child({ props })}
					<Button {...props}>Done</Button>
				{/snippet}
			</Dialog.Close>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<AlertDialog.Root bind:open={revokeOpen}>
	<AlertDialog.Content onCloseAutoFocus={(event) => revoked && focusCreate(event)}>
		<AlertDialog.Header>
			<AlertDialog.Title>Revoke {revokeTarget?.name}?</AlertDialog.Title>
			<AlertDialog.Description>Requests with it stop working right away.</AlertDialog.Description>
		</AlertDialog.Header>
		<AlertDialog.Footer>
			<AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
			<AlertDialog.Action variant="destructive" onclick={revoke} aria-busy={revoking || undefined}>Revoke</AlertDialog.Action>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>
