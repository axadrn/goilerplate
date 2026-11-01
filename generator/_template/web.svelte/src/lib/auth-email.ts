// The address of the last sign-in or reset link, for the "check your email"
// pages. It lives in memory only, so a reload starts over like the HTML
// pages do once their short-lived cookie is gone.
let current = '';

export const authEmail = {
	get: (): string => current,
	set: (email: string): void => {
		current = email;
	}
};
