<script lang="ts">
	import * as Card from '#lib/components/ui/card/index.js';
	import type { WorkspaceDetail } from '#lib/types.js';
	import MemberRow from './member-row.svelte';

	let {
		apiPath,
		detail,
		onchange,
		onleave
	}: {
		apiPath: string;
		detail: WorkspaceDetail;
		onchange: () => Promise<void>;
		// Called when the person removed their own membership.
		onleave: () => Promise<void>;
	} = $props();
</script>

<Card.Root>
	<Card.Header>
		<Card.Title>Members</Card.Title>
		<Card.Description>Roles apply only inside this workspace.</Card.Description>
	</Card.Header>
	<Card.Content>
		<ul class="divide-y rounded-lg border" aria-label="Members">
			{#each detail.members as member (member.id)}
				<MemberRow {apiPath} {member} permissions={detail.permissions} {onchange} {onleave} />
			{/each}
		</ul>
	</Card.Content>
</Card.Root>
