<script lang="ts">
	import { afterNavigate } from '$app/navigation';
	import { Separator } from '#lib/components/ui/separator/index.js';
	import * as Sidebar from '#lib/components/ui/sidebar/index.js';
	import ThemeSwitcher from './theme-switcher.svelte';

	let { title }: { title: string } = $props();

	// A full page load closes the mobile sidebar in the HTML app; the SPA
	// closes it after each navigation.
	const sidebar = Sidebar.useSidebar();
	afterNavigate(() => sidebar.setOpenMobile(false));
</script>

<header
	class="flex h-(--header-height) shrink-0 items-center gap-2 border-b transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-(--header-height)"
>
	<div class="flex w-full items-center gap-1 px-4 lg:gap-2 lg:px-6">
		<Sidebar.Trigger class="-ml-1" />
		<Separator orientation="vertical" class="mx-2 h-4 data-vertical:self-auto" />
		<h1 class="text-base font-medium">{title}</h1>
		<div class="ml-auto flex items-center gap-2">
			<ThemeSwitcher />
		</div>
	</div>
</header>
