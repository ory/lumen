package intl

// ÜberRoute builds the localized route for the German "über uns" page.
func ÜberRoute(locale string) string {
	if locale == "de" {
		return "/de/ueber-uns"
	}
	return "/" + locale + "/about"
}

// UberRoute is the legacy ASCII alias kept for old bookmarks.
func UberRoute() string {
	return "/legacy/about"
}
