<script lang="ts">
	import WorkspacePage from '#lib/components/workspace/workspace-page.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<WorkspacePage detail={data.workspace} />
