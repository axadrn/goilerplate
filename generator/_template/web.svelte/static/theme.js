// Applies the saved theme before the app renders, so a reload never flashes
// the wrong theme. Buttons with data-theme-switcher cycle the theme.
(function () {
	window.themeUtils = window.themeUtils || {};
	const media = window.matchMedia('(prefers-color-scheme: dark)');
	window.themeUtils.getThemePreference = function () {
		return localStorage.getItem('theme') || 'system';
	};
	window.themeUtils.applyTheme = function () {
		const preference = window.themeUtils.getThemePreference();
		const isDark = preference === 'system' ? media.matches : preference === 'dark';
		document.documentElement.classList.toggle('dark', isDark);
		document.dispatchEvent(new CustomEvent('theme-changed'));
	};
	window.themeUtils.cycleTheme = function () {
		const current = window.themeUtils.getThemePreference();
		const next = current === 'system' ? (media.matches ? 'light' : 'dark') : current === 'light' ? 'dark' : 'light';
		localStorage.setItem('theme', next);
		window.themeUtils.applyTheme();
	};
	window.themeUtils.applyTheme();
	media.addEventListener('change', function () {
		if (window.themeUtils.getThemePreference() === 'system') window.themeUtils.applyTheme();
	});
	document.addEventListener('click', function (event) {
		const switcher = event.target.closest('[data-theme-switcher]');
		if (!switcher) return;
		event.preventDefault();
		window.themeUtils.cycleTheme();
	});
})();
