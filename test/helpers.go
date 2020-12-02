package test

import (
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

func ExpectLogs(t *testing.T, logHook *test.Hook, expected []*logrus.Entry) {
	t.Helper()

	if logCount := len(logHook.AllEntries()); logCount != len(expected) {
		for i, entry := range logHook.AllEntries() {
			fmt.Printf("%d - %v\n", i, *entry)
		}

		t.Errorf("expected %d log entries, got %d", len(expected), logCount)
	} else {
		for i, entry := range logHook.AllEntries() {
			expectedEntry := expected[i]
			if entry.Level != expectedEntry.Level {
				t.Errorf("expected entry %d to have level %s, had %s",
					i, expectedEntry.Level, entry.Level)
			}
			if entry.Message != expectedEntry.Message {
				t.Errorf("expected entry %d to have message %q had %q",
					i, expectedEntry.Message, entry.Message)
			}
		}
	}
}

func ExpectLastLog(t *testing.T, logHook *test.Hook, level logrus.Level, msg string) {
	t.Helper()

	if le := logHook.LastEntry(); le == nil {
		t.Errorf("expected a log event, got none")
	} else if le.Level != level {
		t.Errorf("expected a %v level event logevent, got %v", level, le.Level)
	} else if le.Message != msg {
		t.Errorf("expected log message %q got %q", msg, le.Message)
	}
}
