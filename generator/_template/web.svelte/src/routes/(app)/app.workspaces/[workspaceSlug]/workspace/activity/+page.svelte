<script lang="ts">
	import ActivityPage from '#lib/components/workspace/activity-page.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<ActivityPage workspace={data.workspace.workspace} first={data.first} />
