package models

// URLCount is a simple model for representing how many times a given URL has
// been seen.
type URLCount struct {
	// URL is the string representation of the URL.
	URL string
	// Occurrences is a count of how many times the URL has been seen.
	Occurrences int
}
