// Toasts with the titles the HTML pages use.
import { toast } from 'svelte-sonner';
import { ApiError } from './api.js';

export function notifySuccess(title: string, description: string): void {
	toast.success(title, { description });
}

// duration is in milliseconds; the HTML pages keep longer notices for 8 seconds.
export function notifyInfo(title: string, description: string, duration?: number): void {
	toast.info(title, { description, duration });
}

export function notifyError(title: string, description: string): void {
	toast.error(title, { description });
}

// notifyFailure shows a failed request. Rate limits keep the server's title
// for them. Everything else uses the given title with the given description,
// or the server's message without one.
export function notifyFailure(error: unknown, title = 'Error', description?: string): void {
	if (error instanceof ApiError && error.status === 429) {
		notifyError('Too many requests', error.message);
		return;
	}
	if (description) {
		notifyError(title, description);
		return;
	}
	notifyError(title, error instanceof ApiError ? error.message : 'Please try again.');
}
