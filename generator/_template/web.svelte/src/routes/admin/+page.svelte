<script lang="ts">
	import AdminDashboard from '#lib/components/admin/admin-dashboard.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<AdminDashboard dashboard={data.dashboard} />
