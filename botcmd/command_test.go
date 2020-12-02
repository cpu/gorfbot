package botcmd

import (
	"flag"
	"testing"
)

func TestParseFlags(t *testing.T) {
	testCases := []struct {
		name           string
		text           string
		expectedResult string
		expectFoo      bool
	}{
		{
			name:           "bad_parse_example",
			text:           "-aaaa whatever",
			expectedResult: `bad_parse_example: failed to parse "-aaaa whatever": flag provided but not defined: -aaaa`,
		},
		{
			name:           "help_example",
			text:           "-help whatever",
			expectedResult: ":speech_balloon: :bookmark_tabs: Usage of !*help_example*:\n\t`-foo`\tenable foobar (Default: `-foo false`)\n", //nolint:lll
		},
		{
			name: "normal parse",
			text: "-foo whatever",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			flagSet := flag.NewFlagSet(tc.name, flag.ContinueOnError)
			foo := flagSet.Bool("foo", false, "enable foobar")
			result := ParseFlags(tc.text, flagSet)
			if result != tc.expectedResult {
				t.Errorf("expected result %q got %q", tc.expectedResult, result)
			}
			if tc.expectFoo && !*foo {
				t.Errorf("expected -foo was %v got %v", tc.expectFoo, foo)
			}
		})
	}
}
