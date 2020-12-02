package models

import "fmt"

// Emoji is a model for storing a count of emoji/reactjion usage for a user ID.
type Emoji struct {
	// User ID of the user that used the emoji/reaction.
	User string
	// Emoji is the emoji that was used **with** delimiters (e.g. ":wave:" not
	// "wave")
	Emoji string
	// Count is the number of times the emoji/reaction has been used by the user.
	Count int
	// Reaction is true if the model represents a count of **reaction** emoji
	// usage and not usage in messages. When Reaction is false the model
	// represents a count of emoji usage in regular messages and not reactions.
	Reaction bool
}

// String returns a simple representation of the model mostly useful for
// debugging.
func (e Emoji) String() string {
	if e.Reaction {
		return fmt.Sprintf("User %q has reacted with emoji %q %d times",
			e.User, e.Emoji, e.Count)
	}

	return fmt.Sprintf("User %q has used emoji %q in a message %d times",
		e.User, e.Emoji, e.Count)
}
