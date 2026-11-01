<script lang="ts">
	import SettingsPage from '#lib/components/settings/settings-page.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<SettingsPage sessions={data.sessions} tokens={data.tokens} tab={data.tab} passwordRemoved={data.passwordRemoved} />
