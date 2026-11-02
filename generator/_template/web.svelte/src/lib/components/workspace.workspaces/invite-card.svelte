<script lang="ts">
	import { api, ApiError } from '#lib/api.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { notifyWorkspaceFailure, notifyWorkspaceUpdated, roleLabels } from '#lib/workspace/workspace.js';

	let {
		apiPath,
		billingPath,
		full,
		onchange
	}: { apiPath: string; billingPath: string; full: boolean; onchange: () => Promise<void> } = $props();

	const roleItems = (['member', 'admin'] as const).map((value) => ({ value, label: roleLabels[value] }));

	let email = $state('');
	let role = $state('member');
	let invalid = $state(false);
	let pending = $state(false);
	let input = $state<HTMLInputElement | null>(null);

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (pending) {
			return;
		}
		pending = true;
		try {
			await api.post(`${apiPath}/invitations`, { body: { email, role } });
			email = '';
			role = 'member';
			invalid = false;
			notifyWorkspaceUpdated('Invitation sent.');
			await onchange();
		} catch (cause) {
			invalid = cause instanceof ApiError && Boolean(cause.fields.email);
			notifyWorkspaceFailure(cause);
			input?.focus();
		} finally {
			pending = false;
		}
	}
</script>

<Card.Root>
	<Card.Header>
		<Card.Title>Invite a team member</Card.Title>
		<Card.Description>
			{#if full}
				Every seat is taken. <a href={billingPath} class="font-medium text-foreground underline underline-offset-4"
					>Upgrade to invite teammates</a
				>.
			{:else}
				Invitations expire after seven days. Pending invitations do not consume seats.
			{/if}
		</Card.Description>
	</Card.Header>
	<Card.Content>
		<form
			method="POST"
			class="grid gap-3 sm:grid-cols-[1fr_10rem_auto] sm:items-end"
			onsubmit={submit}
			aria-busy={pending || undefined}
		>
			<Field.Field>
				<Field.Label for="invite-email">Email</Field.Label>
				<Input
					bind:ref={input}
					bind:value={email}
					id="invite-email"
					name="email"
					type="email"
					placeholder="teammate@example.com"
					aria-invalid={invalid ? 'true' : 'false'}
					disabled={full}
				/>
			</Field.Field>
			<Field.Field>
				<Field.Label for="invite-role">Role</Field.Label>
				<Select.Root type="single" name="role" bind:value={role} items={roleItems} disabled={full}>
					<Select.Trigger id="invite-role" class="w-full">
						<Select.Value />
					</Select.Trigger>
					<Select.Content align="start">
						{#each roleItems as item (item.value)}
							<Select.Item value={item.value} label={item.label} />
						{/each}
					</Select.Content>
				</Select.Root>
			</Field.Field>
			<Button type="submit" disabled={full}>Send invite</Button>
		</form>
	</Card.Content>
</Card.Root>
