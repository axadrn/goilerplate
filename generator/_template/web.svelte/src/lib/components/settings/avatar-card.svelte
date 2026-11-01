<script lang="ts">
	import Trash2Icon from '@lucide/svelte/icons/trash-2';
	import UploadIcon from '@lucide/svelte/icons/upload';
	import { tick } from 'svelte';
	import { api, ApiError } from '#lib/api.js';
	import * as AlertDialog from '#lib/components/ui/alert-dialog/index.js';
	import * as Avatar from '#lib/components/ui/avatar/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { notifyError, notifyFailure, notifySuccess } from '#lib/notify.js';
	import { session } from '#lib/session.svelte.js';
	import type { Account, AvatarAnswer } from '#lib/types.js';

	let { account }: { account: Account } = $props();

	let files = $state<FileList | undefined>();
	let invalid = $state(false);
	let uploading = $state(false);
	let removeOpen = $state(false);
	let removing = $state(false);
	let input = $state<HTMLInputElement | null>(null);

	const initial = $derived(account.user.email.charAt(0).toUpperCase());

	function setAvatar(url: string | null) {
		if (session.account) {
			session.account.avatar_url = url;
		}
	}

	async function reject(description: string) {
		invalid = true;
		notifyError('Error', description);
		await tick();
		input?.focus();
	}

	async function upload(event: SubmitEvent) {
		event.preventDefault();
		if (uploading) {
			return;
		}
		const file = files?.[0];
		if (!file) {
			await reject('No file uploaded');
			return;
		}
		uploading = true;
		const form = new FormData();
		form.append('file', file);
		try {
			const answer = await api.post<AvatarAnswer>('/api/v1/account/avatar', { form });
			setAvatar(answer.avatar_url);
			invalid = false;
			files = undefined;
			if (input) {
				input.value = '';
			}
			notifySuccess('Success', 'Avatar uploaded successfully');
		} catch (cause) {
			if (cause instanceof ApiError && cause.fields.file) {
				await reject(cause.fields.file);
			} else {
				notifyFailure(cause, 'Error', 'Failed to upload avatar');
			}
		} finally {
			uploading = false;
		}
	}

	async function remove() {
		if (removing) {
			return;
		}
		removing = true;
		try {
			await api.delete('/api/v1/account/avatar');
			setAvatar(null);
			notifySuccess('Success', 'Avatar removed successfully');
		} catch (cause) {
			notifyFailure(cause, 'Error', 'Failed to delete avatar');
		} finally {
			removing = false;
			removeOpen = false;
		}
	}
</script>

<Card.Root>
	<Card.Header>
		<Card.Title>Avatar</Card.Title>
		<Card.Description>Upload a profile picture (Max 5MB, JPG/PNG/WebP)</Card.Description>
	</Card.Header>
	<Card.Content>
		<div class="flex flex-col items-start gap-4 md:flex-row md:gap-6">
			<div id="settings-avatar">
				<Avatar.Root class="size-20">
					{#if account.avatar_url}
						<Avatar.Image src={account.avatar_url} alt={account.user.email} />
					{/if}
					<Avatar.Fallback>{initial}</Avatar.Fallback>
				</Avatar.Root>
			</div>
			<div class="w-full flex-1 space-y-4">
				<form id="avatar-upload-form" class="space-y-4" onsubmit={upload} aria-busy={uploading || undefined}>
					<div class="space-y-2">
						<Label for="avatar">Choose Image</Label>
						<Input
							bind:ref={input}
							bind:files
							type="file"
							id="avatar"
							name="file"
							accept="image/jpeg,image/jpg,image/png,image/webp"
							aria-invalid={invalid ? 'true' : 'false'}
							aria-describedby="avatar-hint"
						/>
						<p id="avatar-hint" class="text-sm text-muted-foreground">JPEG, PNG or WebP. Max file size 5MB.</p>
					</div>
					<div class="flex flex-col gap-2 sm:flex-row sm:justify-end">
						<Button type="submit" class="w-full sm:w-auto">
							<UploadIcon class="mr-2 size-4" />
							Upload Avatar
						</Button>
						<AlertDialog.Root bind:open={removeOpen}>
							<AlertDialog.Trigger>
								{#snippet child({ props })}
									<Button variant="outline" class="w-full sm:w-auto" {...props}>
										<Trash2Icon class="mr-2 size-4" />
										Remove Avatar
									</Button>
								{/snippet}
							</AlertDialog.Trigger>
							<AlertDialog.Content>
								<AlertDialog.Header>
									<AlertDialog.Title>Remove Avatar</AlertDialog.Title>
									<AlertDialog.Description>
										Are you sure you want to remove your avatar? This action cannot be undone.
									</AlertDialog.Description>
								</AlertDialog.Header>
								<AlertDialog.Footer>
									<AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
									<AlertDialog.Action onclick={remove} aria-busy={removing || undefined}>Remove</AlertDialog.Action>
								</AlertDialog.Footer>
							</AlertDialog.Content>
						</AlertDialog.Root>
					</div>
				</form>
			</div>
		</div>
	</Card.Content>
</Card.Root>
