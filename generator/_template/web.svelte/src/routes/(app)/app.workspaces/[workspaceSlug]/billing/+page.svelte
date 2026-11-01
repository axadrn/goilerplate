<script lang="ts">
	import BillingPage from '#lib/components/billing/billing-page.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<BillingPage
	billing={data.billing}
	description="Manage billing and seats for {data.workspace.workspace.name}"
	workspaceSlug={data.workspace.workspace.slug}
/>
