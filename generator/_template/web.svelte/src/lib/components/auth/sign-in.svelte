<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { config } from '#lib/config.js';
	import AuthCard from './auth-card.svelte';
	import MagicLinkForm from './magic-link-form.svelte';
	import OAuthButtons from './oauth-buttons.svelte';

	// error is shown under the email field, like a failed sign-in link.
	let { error = '' }: { error?: string } = $props();
</script>

<AuthCard title="Welcome" description="Sign in or create your account">
	<Field.Group>
		<MagicLinkForm {error} />
		{#if config.google || config.github}
			<Field.Separator class="*:data-[slot=field-separator-content]:bg-card">Or continue with</Field.Separator>
			<OAuthButtons />
		{/if}
		<Field.Separator class="*:data-[slot=field-separator-content]:bg-card">Or</Field.Separator>
		<Button href="/auth/password" variant="outline" class="w-full">Sign in with password</Button>
	</Field.Group>
</AuthCard>
