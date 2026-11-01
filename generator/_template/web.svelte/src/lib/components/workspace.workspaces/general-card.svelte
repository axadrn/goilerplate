<script lang="ts">
	import { api } from '#lib/api.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { notifyWorkspaceFailure, notifyWorkspaceUpdated } from '#lib/workspace/workspace.js';

	let { apiPath, name: savedName, onchange }: { apiPath: string; name: string; onchange: () => Promise<void> } = $props();

	let name = $derived(savedName);
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
			await api.patch(`${apiPath}/name`, { body: { name } });
			invalid = false;
			notifyWorkspaceUpdated('Workspace name updated.');
			await onchange();
		} catch (cause) {
			invalid = true;
			notifyWorkspaceFailure(cause);
			input?.focus();
		} finally {
			pending = false;
		}
	}
</script>

<Card.Root>
	<Card.Header>
		<Card.Title>General</Card.Title>
		<Card.Description>The slug in workspace URLs stays stable when the name changes.</Card.Description>
	</Card.Header>
	<Card.Content>
		<form
			method="POST"
			class="flex flex-col gap-3 sm:flex-row sm:items-end"
			onsubmit={submit}
			aria-busy={pending || undefined}
		>
			<Field.Field class="flex-1">
				<Field.Label for="workspace-name">Workspace name</Field.Label>
				<Input
					bind:ref={input}
					bind:value={name}
					id="workspace-name"
					name="name"
					aria-invalid={invalid ? 'true' : 'false'}
				/>
			</Field.Field>
			<Button type="submit">Save</Button>
		</form>
	</Card.Content>
</Card.Root>
