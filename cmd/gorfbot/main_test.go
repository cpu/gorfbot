package main

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestStringToLevel(t *testing.T) {
	testCases := []struct {
		levelStr string
		level    logrus.Level
	}{
		{
			levelStr: "ErRoR", // Test case insensitivity
			level:    logrus.ErrorLevel,
		},
		{
			levelStr: "error",
			level:    logrus.ErrorLevel,
		},
		{
			levelStr: "warn",
			level:    logrus.WarnLevel,
		},
		{
			levelStr: "info",
			level:    logrus.InfoLevel,
		},
		{
			levelStr: "debug",
			level:    logrus.DebugLevel,
		},
		{
			levelStr: "trace",
			level:    logrus.TraceLevel,
		},
		{
			levelStr: "magenta", // Test default for unknown levels
			level:    logrus.WarnLevel,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.levelStr, func(t *testing.T) {
			if actual := stringToLevel(tc.levelStr); actual != tc.level {
				t.Errorf("expected level str %q to be %v, was %v",
					tc.levelStr, tc.level, actual)
			}
		})
	}
}
