//nolint:funlen
package reactjikeys

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/cpu/gorfbot/botcmd"
	"github.com/cpu/gorfbot/config"
	"github.com/cpu/gorfbot/test"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

func TestConfigure(t *testing.T) {
	log, _ := logtest.NewNullLogger()
	testConfig := &config.Config{
		ReactjiKeysConf: config.ReactjiKeysConfig{
			Keywords: map[string][]string{
				"test": {"thumbsup", "thumbsdown"},
			},
		},
	}
	cmd := &reactjikeysPattern{}

	if err := cmd.Configure(log, testConfig); err != nil {
		t.Errorf("expected no err from configure, got %v", err)
	} else if cmd.log != log {
		t.Errorf("expected log to be set to %p was %p", log, cmd.log)
	} else if !reflect.DeepEqual(cmd.config, testConfig.ReactjiKeysConf) {
		t.Errorf("expected config to be set to %#v was %#v",
			testConfig.ReactjiKeysConf, cmd.config)
	}

	// Loading a nil config shouldn't err either
	cmd = &reactjikeysPattern{}
	emptyConfig := config.ReactjiKeysConfig{}

	if err := cmd.Configure(log, nil); err != nil {
		t.Errorf("expected no err from configure, got %v", err)
	} else if cmd.log != log {
		t.Errorf("expected log to be set to %p was %p", log, cmd.log)
	} else if !reflect.DeepEqual(cmd.config, emptyConfig) {
		t.Errorf("expected config to be set to empty was %q", cmd.config)
	}
}

func setup(conf config.ReactjiKeysConfig) (*logtest.Hook, *reactjikeysPattern, botcmd.RunContext) {
	log, logHook := logtest.NewNullLogger()
	cmd := &reactjikeysPattern{
		log:    log,
		config: conf,
	}

	ctx := botcmd.RunContext{}

	return logHook, cmd, ctx
}

func TestRunTooFewMatches(t *testing.T) {
	_, cmd, ctx := setup(config.ReactjiKeysConfig{})

	// Too few submatches
	if _, err := cmd.Run([][]string{}, ctx); err == nil {
		t.Errorf("expected err from Run with empty matches, got nil")
	}
}

func TestRunTooSmallMatch(t *testing.T) {
	_, cmd, ctx := setup(config.ReactjiKeysConfig{})

	// Too small submatch
	if _, err := cmd.Run([][]string{{"a"}}, ctx); err == nil {
		t.Errorf("expected err from Run with empty matches, got nil")
	}
}

func TestRun(t *testing.T) {
	conf := config.ReactjiKeysConfig{
		Keywords: map[string][]string{
			"hello":   {"wave"},
			"goodbye": {"wave", "cry"},
		},
	}
	logHook, cmd, ctx := setup(conf)

	repeatWord := func(word string, times int) []string {
		var results []string
		for i := 0; i < times; i++ {
			results = append(results, word)
		}

		return results
	}
	repeatSubmatches := func(word string, times int) [][]string {
		var results [][]string
		for i := 0; i < times; i++ {
			results = append(results, repeatWord(word, 2))
		}

		return results
	}

	testCases := []struct {
		name              string
		allSubmatches     [][]string
		expectedHits      []string
		expectedReactions []string
	}{
		{
			name:          "no matches",
			allSubmatches: [][]string{{"a", "a"}},
		},
		{
			name:              "one match",
			allSubmatches:     [][]string{{"hello", "hello"}},
			expectedHits:      []string{"hello"},
			expectedReactions: []string{"wave"},
		},
		{
			name:              "two distinct matches",
			allSubmatches:     [][]string{{"hello", "hello"}, {"goodbye", "goodbye"}},
			expectedHits:      []string{"hello", "goodbye"},
			expectedReactions: []string{"cry", "wave"}, // note: just one wave, not two!
		},
		{
			name:              "mixed case match",
			allSubmatches:     [][]string{{"HeLLo", "HeLLo"}},
			expectedHits:      []string{"hello"},
			expectedReactions: []string{"wave"},
		},
		{
			name:              "repeated matches",
			allSubmatches:     repeatSubmatches("hello", 5),
			expectedHits:      repeatWord("hello", 5),
			expectedReactions: []string{"wave"}, // note: just one wave, not five!
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if res, err := cmd.Run(tc.allSubmatches, botcmd.RunContext{}); err != nil {
				t.Fatalf("unexpected err from Run: %v", err)
			} else if res.Message != "" {
				t.Errorf("unexpected result message from Run: %q", res.Message)
			} else if !reflect.DeepEqual(res.Reactji, tc.expectedReactions) {
				t.Errorf("expected result reactions %#v got %#v",
					tc.expectedReactions, res.Reactji)
			}

			var expectedLogs []*logrus.Entry
			for _, expectedHit := range tc.expectedHits {
				expectedLogMsg := fmt.Sprintf("word: %q triggers %#v",
					expectedHit, conf.Keywords[expectedHit])
				expectedLogs = append(expectedLogs, &logrus.Entry{
					Level:   logrus.InfoLevel,
					Message: expectedLogMsg,
				})
			}
			test.ExpectLogs(t, logHook, expectedLogs)
			logHook.Reset()
		})
	}

	if _, err := cmd.Run([][]string{{"a"}}, ctx); err == nil {
		t.Errorf("expected err from Run with empty matches, got nil")
	}
}
