<script lang="ts">
	import ActivityIcon from '@lucide/svelte/icons/activity';
	import { tick } from 'svelte';
	import { api } from '#lib/api.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import * as Item from '#lib/components/ui/item/index.js';
	import { formatDayTime } from '#lib/format.js';
	import { notifyFailure } from '#lib/notify.js';
	import type { ActivityPage, Workspace } from '#lib/types.js';
	import { activityLabel } from '#lib/workspace/workspace.js';

	let { workspace, first }: { workspace: Workspace; first: ActivityPage } = $props();

	let events = $derived(first.items);
	let cursor = $derived(first.next_cursor);
	let pending = $state(false);
	let list = $state<HTMLElement | null>(null);

	// loadMore appends the next page and moves focus to its first event.
	async function loadMore() {
		if (pending || !cursor) {
			return;
		}
		pending = true;
		try {
			const next = await api.get<ActivityPage>(`/api/v1/workspaces/${encodeURIComponent(workspace.slug)}/activity`, {
				query: { cursor }
			});
			const firstNew = events.length;
			events = [...events, ...next.items];
			cursor = next.next_cursor;
			await tick();
			list?.querySelectorAll<HTMLElement>('[data-event]')[firstNew]?.focus();
		} catch (cause) {
			notifyFailure(cause, 'Activity unavailable', 'Workspace activity could not be loaded. Please try again.');
		} finally {
			pending = false;
		}
	}
</script>

<div class="container max-w-5xl space-y-6 px-6 py-8">
	<div>
		<h1 class="text-3xl font-bold">Activity</h1>
		<p class="mt-2 text-muted-foreground">Security and member changes for {workspace.name}.</p>
	</div>
	<Card.Root>
		<Card.Content>
			{#if events.length === 0}
				<Empty.Root>
					<Empty.Header>
						<Empty.Media variant="icon">
							<ActivityIcon />
						</Empty.Media>
						<Empty.Title>No activity yet</Empty.Title>
						<Empty.Description>Security and member changes will appear here.</Empty.Description>
					</Empty.Header>
				</Empty.Root>
			{:else}
				<div bind:this={list}>
					<Item.Group class="gap-2" aria-label="Workspace activity">
						{#each events as event (event.id)}
							<Item.Root size="sm" tabindex={-1} data-event={event.id}>
								<Item.Content>
									<Item.Title>{activityLabel(event.action)}</Item.Title>
									<Item.Description>{event.actor_email || 'System'}</Item.Description>
								</Item.Content>
								<Item.Actions>
									<time class="text-sm text-muted-foreground" datetime={event.created_at}>
										{formatDayTime(event.created_at)}
									</time>
								</Item.Actions>
							</Item.Root>
						{/each}
					</Item.Group>
				</div>
				{#if cursor}
					<div class="mt-4 flex justify-center">
						<Button variant="outline" aria-busy={pending || undefined} onclick={loadMore}>Load more</Button>
					</div>
				{/if}
			{/if}
		</Card.Content>
	</Card.Root>
</div>
