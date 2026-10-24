/* Initialise Pagefind UI in the sidebar widget and on the /search/ page. */
(function () {
	function boot() {
		if (typeof PagefindUI === 'undefined') { return; }
		var opts = {
			showSubResults: true,
			showImages: false,
			resetStyles: false,
			pageSize: 8,
			translations: { placeholder: 'Search TorBox…' }
		};
		var side = document.getElementById('search');
		if (side) { new PagefindUI(Object.assign({ element: side }, opts)); }
		var full = document.getElementById('search-page');
		if (full) {
			var ui = new PagefindUI(Object.assign({ element: full, autofocus: true }, opts));
			var q = new URLSearchParams(location.search).get('q');
			if (q) { ui.triggerSearch(q); }
		}
	}
	if (document.readyState === 'loading') { document.addEventListener('DOMContentLoaded', boot); } else { boot(); }
})();
