<script lang="ts">
	import AuthCard from '#lib/components/auth/auth-card.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<AuthCard title="Check your email" description="We sent a reset link to {data.email}">
	<Field.Field>
		<Button href="/auth" variant="outline" class="w-full">Back to sign in</Button>
	</Field.Field>
</AuthCard>
