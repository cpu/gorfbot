package slack

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

//nolint:funlen
func TestParseTimestamp(t *testing.T) {
	testCases := []struct {
		name            string
		ts              string
		expectedWarnMsg string
		expectedErr     string
		expectedTime    time.Time
	}{
		{
			name:         "valid TS, with UUID",
			ts:           "1607050812.008800",
			expectedTime: time.Unix(1607050812, 0),
		},
		{
			name:         "valid TS, no UUID",
			ts:           "1607050812",
			expectedTime: time.Unix(1607050812, 0),
		},
		{
			name:            "weird TS, with UUID",
			ts:              "1607050812.008800.abcd",
			expectedWarnMsg: `found weird timestamp "1607050812.008800.abcd", components [1607050812 008800 abcd]`,
			expectedTime:    time.Unix(1607050812, 0),
		},
		{
			name:        "invalid TS, no UUID",
			ts:          "abcdef",
			expectedErr: `unable to convert Slack timestamp "abcdef" to time.Time`,
		},
		{
			name:        "invalid TS, with UUID",
			ts:          "abcdef.008800",
			expectedErr: `unable to convert Slack timestamp "abcdef.008800" to time.Time`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			log, logHook := test.NewNullLogger()
			defer logHook.Reset()

			client := clientImpl{
				log: log,
			}

			if actual, err := client.ParseTimestamp(tc.ts); err != nil {
				if tc.expectedErr == "" {
					t.Errorf("expected no err parsing %q, got %v", tc.ts, err)
				} else if tc.expectedErr != err.Error() {
					t.Errorf("expected err %q got %q", tc.expectedErr, err.Error())
				}
			} else {
				if tc.expectedTime != actual {
					t.Errorf("expected time %s from parsing %q got %s",
						tc.expectedTime, tc.ts, actual)
				}
			}
			if tc.expectedWarnMsg != "" {
				if le := logHook.LastEntry(); le == nil {
					t.Errorf("expected a log event, got none")
				} else if le.Level != logrus.WarnLevel {
					t.Errorf("expected a warn level logevent, got %v", le.Level)
				} else if le.Message != tc.expectedWarnMsg {
					t.Errorf("expected specific warn message, got %q", le.Message)
				}
			}
		})
	}
}
