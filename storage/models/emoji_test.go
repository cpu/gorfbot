package models

import "testing"

func TestEmojiString(t *testing.T) {
	testCases := []struct {
		name     string
		e        Emoji
		expected string
	}{
		{
			name: "emoji, not reaction",
			e: Emoji{
				User:  "U000",
				Emoji: ":wave:",
				Count: 10,
			},
			expected: `User "U000" has used emoji ":wave:" in a message 10 times`,
		},
		{
			name: "emoji, reaction",
			e: Emoji{
				User:     "U000",
				Emoji:    ":wave:",
				Count:    10,
				Reaction: true,
			},
			expected: `User "U000" has reacted with emoji ":wave:" 10 times`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if actual := tc.e.String(); actual != tc.expected {
				t.Errorf("expected emoji %v to have string form %q, was %q",
					tc.e, tc.expected, actual)
			}
		})
	}
}
