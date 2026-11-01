<script lang="ts">
	import CreditCardIcon from '@lucide/svelte/icons/credit-card';
	import SettingsIcon from '@lucide/svelte/icons/settings';
	import ShieldIcon from '@lucide/svelte/icons/shield';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import { session } from '#lib/session.svelte.js';
</script>

<DropdownMenu.Item>
	{#snippet child({ props })}
		<a href="/app/settings" {...props}>
			<span class="flex items-center">
				<SettingsIcon class="mr-2 size-4" />
				Settings
			</span>
		</a>
	{/snippet}
</DropdownMenu.Item>
{#if session.account?.is_admin}
	<DropdownMenu.Item>
		{#snippet child({ props })}
			<a href="/admin" {...props}>
				<span class="flex items-center">
					<ShieldIcon class="mr-2 size-4" />
					Admin
				</span>
			</a>
		{/snippet}
	</DropdownMenu.Item>
{/if}
<DropdownMenu.Item>
	{#snippet child({ props })}
		<a href="/app/billing" {...props}>
			<span class="flex items-center">
				<CreditCardIcon class="mr-2 size-4" />
				Billing
			</span>
		</a>
	{/snippet}
</DropdownMenu.Item>
