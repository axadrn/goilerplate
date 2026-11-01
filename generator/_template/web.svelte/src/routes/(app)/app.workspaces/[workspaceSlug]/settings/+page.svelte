<script lang="ts">
	import SettingsPage from '#lib/components/settings/settings-page.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<!-- New API tokens are bound to this workspace unless the person picks another. -->
<SettingsPage
	sessions={data.sessions}
	tokens={data.tokens}
	tab={data.tab}
	passwordRemoved={data.passwordRemoved}
	workspaceId={data.workspace.workspace.id}
/>
