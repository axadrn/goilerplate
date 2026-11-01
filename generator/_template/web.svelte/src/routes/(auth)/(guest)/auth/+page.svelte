<script lang="ts">
	import SignIn from '#lib/components/auth/sign-in.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<SignIn error={data.error} />
