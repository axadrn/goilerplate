// The only way the app talks to the Go server. Requests go to the same
// origin with the session cookie. Requests that change data carry the CSRF
// token from GET /api/v1/csrf.
import type { CSRFToken } from './types.js';

type Method = 'GET' | 'POST' | 'PATCH' | 'DELETE';

interface ErrorEnvelope {
	error?: { code?: string; message?: string; fields?: Record<string, string> };
}

export interface RequestOptions {
	// Sent as JSON.
	body?: unknown;
	// Sent as multipart/form-data, for file uploads.
	form?: FormData;
	query?: Record<string, string | number | boolean | undefined>;
	signal?: AbortSignal;
	// The fetch a SvelteKit load function receives. Defaults to window.fetch.
	fetch?: typeof fetch;
}

// ApiError carries the error envelope of a non-2xx answer: the status, a
// stable code, a message for people and the messages per invalid field.
export class ApiError extends Error {
	readonly status: number;
	readonly code: string;
	readonly fields: Record<string, string>;

	constructor(status: number, code: string, message: string, fields: Record<string, string> = {}) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
		this.code = code;
		this.fields = fields;
	}
}

let csrfToken: Promise<string> | null = null;

function csrf(): Promise<string> {
	csrfToken ??= request<CSRFToken>('GET', '/api/v1/csrf').then(
		(answer) => answer.token,
		(error: unknown) => {
			csrfToken = null;
			throw error;
		}
	);
	return csrfToken;
}

export async function request<T>(method: Method, path: string, options: RequestOptions = {}): Promise<T> {
	const response = await send(method, path, options);
	if (response.status === 204) {
		return undefined as T;
	}
	const body: unknown = await response.json().catch(() => null);
	if (!response.ok) {
		throw toApiError(response.status, body);
	}
	return body as T;
}

async function send(method: Method, path: string, options: RequestOptions): Promise<Response> {
	const changesData = method !== 'GET';
	const cachedToken = csrfToken !== null;
	const response = await fetchOnce(method, path, options, changesData ? await csrf() : undefined);
	// The CSRF cookie can change under a cached token, for example after
	// another tab signed in. Fetch a fresh token once and try again.
	if (changesData && cachedToken && response.status === 403) {
		csrfToken = null;
		return fetchOnce(method, path, options, await csrf());
	}
	return response;
}

async function fetchOnce(method: Method, path: string, options: RequestOptions, token?: string): Promise<Response> {
	const headers = new Headers({ Accept: 'application/json' });
	let body: BodyInit | undefined;
	if (options.form) {
		body = options.form;
	} else if (options.body !== undefined) {
		headers.set('Content-Type', 'application/json');
		body = JSON.stringify(options.body);
	}
	if (token) {
		headers.set('X-CSRF-Token', token);
	}
	try {
		return await (options.fetch ?? fetch)(url(path, options.query), {
			method,
			headers,
			body,
			credentials: 'same-origin',
			signal: options.signal
		});
	} catch (error) {
		if (error instanceof DOMException && error.name === 'AbortError') {
			throw error;
		}
		throw new ApiError(0, 'network_error', 'The server could not be reached. Check your connection and try again.');
	}
}

function url(path: string, query: RequestOptions['query']): string {
	const search = new URLSearchParams();
	for (const [key, value] of Object.entries(query ?? {})) {
		if (value !== undefined) {
			search.set(key, String(value));
		}
	}
	const encoded = search.toString();
	return encoded ? `${path}?${encoded}` : path;
}

function toApiError(status: number, body: unknown): ApiError {
	const detail = (body as ErrorEnvelope | null)?.error;
	if (detail?.code && detail.message) {
		return new ApiError(status, detail.code, detail.message, detail.fields);
	}
	return new ApiError(status, 'internal', 'Something went wrong. Please try again.');
}

export const api = {
	get: <T>(path: string, options?: Omit<RequestOptions, 'body' | 'form'>) => request<T>('GET', path, options),
	post: <T>(path: string, options?: RequestOptions) => request<T>('POST', path, options),
	patch: <T>(path: string, options?: RequestOptions) => request<T>('PATCH', path, options),
	delete: <T>(path: string, options?: RequestOptions) => request<T>('DELETE', path, options)
};
