// Dates in the formats of the HTML pages, in the browser's time zone.
const day = new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
const longDay = new Intl.DateTimeFormat('en-US', { month: 'long', day: 'numeric', year: 'numeric' });
const time = new Intl.DateTimeFormat('en-US', { hour: '2-digit', minute: '2-digit', hourCycle: 'h23' });

// formatDay answers "Jan 2, 2006".
export function formatDay(value: string): string {
	return day.format(new Date(value));
}

// formatDayTime answers "Jan 2, 2006 at 15:04".
export function formatDayTime(value: string): string {
	const date = new Date(value);
	return `${day.format(date)} at ${time.format(date)}`;
}

// formatLongDay answers "January 2, 2006".
export function formatLongDay(value: string): string {
	return longDay.format(new Date(value));
}
