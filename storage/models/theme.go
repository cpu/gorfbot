package models

// Theme is a simple model for storing a Slack theme, its name, and the user ID
// of its creator.
type Theme struct {
	// Name of the theme.
	Name string
	// String representation of the theme (e.g. 8 comma separated hex colours like
	// "#624574,#7C4430,#5F303E,#305F5E,#3F5B32,#62502F,#875566,#3F567A")
	Theme string
	// The user ID of the creator of the theme (note: not the friendly username).
	Creator string
}
