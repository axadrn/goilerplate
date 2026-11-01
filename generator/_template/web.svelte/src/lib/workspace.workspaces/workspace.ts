// Labels, paths and error messages of the workspace pages, as in the HTML
// app.
import { ApiError } from '#lib/api.js';
import { notifyError, notifyFailure, notifySuccess } from '#lib/notify.js';
import type { Account, WorkspaceRole } from '#lib/types.js';

export const roleLabels: Record<WorkspaceRole, string> = { owner: 'Owner', admin: 'Admin', member: 'Member' };

const actionLabels: Record<string, string> = {
	'workspace.created': 'Created the workspace',
	'workspace.updated': 'Updated workspace settings',
	'workspace.invitation.sent': 'Sent an invitation',
	'workspace.invitation.created': 'Created an invitation',
	'workspace.invitation.revoked': 'Revoked an invitation',
	'workspace.invitation.accepted': 'Accepted an invitation',
	'workspace.member.role_changed': 'Changed a member role',
	'workspace.member.removed': 'Removed a member',
	'workspace.member.left': 'Left the workspace',
	'billing.checkout_started': 'Started a billing checkout',
	'billing.management_opened': 'Opened billing management'
};

export function activityLabel(action: string): string {
	return actionLabels[action] ?? action;
}

// firstWorkspacePath answers the dashboard of the account's first
// workspace, or the page that creates one.
export function firstWorkspacePath(account: Account | null): string {
	const first = account?.workspaces?.[0];
	return first ? `/app/${first.workspace.slug}/dashboard` : '/workspaces/new';
}

export function notifyWorkspaceUpdated(description: string): void {
	notifySuccess('Workspace updated', description);
}

// notifyWorkspaceFailure shows the HTML page's toast for a failed workspace
// action.
export function notifyWorkspaceFailure(cause: unknown): void {
	if (!(cause instanceof ApiError)) {
		notifyFailure(cause, 'Action failed', 'The workspace action could not be completed. Please try again.');
		return;
	}
	if (cause.code === 'validation_failed') {
		if (cause.fields.role) {
			notifyError('Invalid role', 'Choose Owner, Admin, or Member where allowed.');
		} else if (cause.fields.name) {
			notifyError('Invalid workspace name', 'Workspace names must contain between 2 and 80 printable characters.');
		} else {
			notifyError('Invalid email', 'Enter a valid email address.');
		}
		return;
	}
	switch (cause.code) {
		case 'forbidden':
			notifyError('Action not allowed', 'You do not have permission to perform this workspace action.');
			return;
		case 'last_owner':
			notifyError('Owner required', 'A workspace must always have at least one owner.');
			return;
		case 'seat_limit':
			notifyError('No seats available', 'Upgrade the workspace or remove a member before inviting someone new.');
			return;
		case 'already_member':
			notifyError('Already a member', 'That user already belongs to this workspace.');
			return;
		case 'active_subscription':
			notifyError(
				'Workspace cannot be deleted',
				'Finish or wait for checkout to expire. Cancel paid subscriptions and wait for access to end before deleting this workspace.'
			);
			return;
		case 'not_found':
			notifyError('Not found', 'The requested member or invitation no longer exists.');
			return;
	}
	notifyFailure(cause, 'Action failed', 'The workspace action could not be completed. Please try again.');
}
