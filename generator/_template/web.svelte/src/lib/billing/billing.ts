// Billing requests and the messages of the HTML billing pages. Without
// workspaces billing belongs to the account, with workspaces to a workspace.
import { error } from '@sveltejs/kit';
import { api, ApiError } from '#lib/api.js';
import { notifyError, notifyFailure } from '#lib/notify.js';
import type { Billing, BillingInterval, Subscription, URLAnswer } from '#lib/types.js';

export interface BillingPaths {
	billing: string;
	checkout: string;
	manage: string;
}

// billingPaths answers the endpoints of the account, or of the workspace
// with the given slug.
export function billingPaths(workspaceSlug?: string): BillingPaths {
	if (!workspaceSlug) {
		return {
			billing: '/api/v1/billing',
			checkout: '/api/v1/billing/checkout',
			manage: '/api/v1/billing/portal'
		};
	}
	const base = `/api/v1/workspaces/${encodeURIComponent(workspaceSlug)}/billing`;
	return { billing: base, checkout: `${base}/checkout`, manage: `${base}/manage` };
}

// loadBilling loads the billing page. Members who may not manage billing get
// the HTML page's 403.
export async function loadBilling(fetcher: typeof fetch, workspaceSlug?: string): Promise<Billing> {
	try {
		return await api.get<Billing>(billingPaths(workspaceSlug).billing, { fetch: fetcher });
	} catch (cause) {
		if (cause instanceof ApiError && cause.status === 403) {
			error(403, { title: 'Access denied', message: 'Only workspace owners can manage billing.' });
		}
		error(500, 'Billing could not be loaded. Please try again.');
	}
}

// startCheckout asks for a provider checkout and answers its URL.
export async function startCheckout(paths: BillingPaths, planId: string, interval: BillingInterval): Promise<string> {
	const answer = await api.post<URLAnswer>(paths.checkout, { body: { plan_id: planId, interval } });
	return answer.url;
}

// manageURL answers the provider portal or billing management URL.
export async function manageURL(paths: BillingPaths): Promise<string> {
	return (await api.get<URLAnswer>(paths.manage)).url;
}

export function notifyCheckoutFailure(cause: unknown): void {
	switch (cause instanceof ApiError ? cause.code : '') {
		case 'forbidden':
			notifyError('Action not allowed', 'Only workspace owners can start checkout.');
			return;
		case 'invalid_plan':
			notifyError('Invalid plan', 'Choose an available paid plan.');
			return;
		case 'invalid_interval':
			notifyError('Invalid billing interval', 'Choose monthly or yearly billing.');
			return;
		case 'checkout_pending':
		case 'checkout_conflict':
		case 'already_paid':
			notifyError('Checkout already active', 'Finish the existing checkout or manage the current subscription first.');
			return;
	}
	notifyFailure(cause, 'Checkout unavailable', 'Checkout could not be started. Please try again.');
}

// notifyManageFailure shows why the portal did not open. unavailable is the
// HTML page's text for a provider failure.
export function notifyManageFailure(cause: unknown, unavailable: string): void {
	if (cause instanceof ApiError && cause.code === 'forbidden') {
		notifyError('Action not allowed', 'Only workspace owners can manage billing.');
		return;
	}
	if (cause instanceof ApiError && cause.code === 'no_billing_account') {
		notifyError('Billing unavailable', cause.message);
		return;
	}
	notifyFailure(cause, 'Billing unavailable', unavailable);
}

const currencySymbols: Record<string, string> = { usd: '$', eur: '€', gbp: '£' };

// formatSubscriptionPrice answers "$29/month" like the HTML page, or an
// empty string without an amount.
export function formatSubscriptionPrice(subscription: Subscription): string {
	if (!subscription.amount) {
		return '';
	}
	const symbol = currencySymbols[subscription.currency] ?? '$';
	const interval = subscription.interval === 'yearly' ? 'year' : 'month';
	return `${symbol}${Math.round(subscription.amount / 100)}/${interval}`;
}

// hasBillingAccount reports whether the payment provider knows a customer
// for the subscription. The provider is set together with the customer.
export function hasBillingAccount(subscription: Subscription | null): boolean {
	return Boolean(subscription?.provider);
}
