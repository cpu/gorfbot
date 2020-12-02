package botcmd

import (
	"reflect"
	"regexp"
	"testing"

	"github.com/cpu/gorfbot/config"
	"github.com/cpu/gorfbot/slack"
	"github.com/sirupsen/logrus"
)

type mockCmdHandler struct{}

func (mh mockCmdHandler) Configure(l *logrus.Logger, c *config.Config) error {
	return nil
}

func (mh mockCmdHandler) Run(text string, ctx RunContext) (RunResult, error) {
	return RunResult{}, nil
}

type mockPatternHandler struct{}

func (mh mockPatternHandler) Configure(l *logrus.Logger, c *config.Config) error {
	return nil
}

func (mh mockPatternHandler) Run(allSubmatches [][]string, ctx RunContext) (RunResult, error) {
	return RunResult{}, nil
}

type mockReactionHandler struct{}

func (mrh mockReactionHandler) Configure(l *logrus.Logger, c *config.Config) error {
	return nil
}

func (mrh mockReactionHandler) Run(reacton *slack.Reaction, ctx RunContext) error {
	return nil
}

func TestCommandRegistryAddCommand(t *testing.T) {
	registry := NewRegistry()

	testCases := []struct {
		name     string
		command  *BasicCommand
		expected bool
	}{
		{
			name: "nil command",
		},
		{
			name:    "empty name",
			command: &BasicCommand{},
		},
		{
			name:    "nil handler",
			command: &BasicCommand{Name: "nohandler"},
		},
		{
			name: "valid handler",
			command: &BasicCommand{
				Name:    "valid",
				Handler: mockCmdHandler{},
			},
			expected: true,
		},
		{
			name: "duplicate name",
			command: &BasicCommand{
				Name:    "valid",
				Handler: mockCmdHandler{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if actual := registry.AddCommand(tc.command); actual != tc.expected {
				t.Errorf("expected AddCommand(%v) to return %v got %v",
					tc.command, tc.expected, actual)
			}
		})
	}
}

func TestCommandRegistryAddPattern(t *testing.T) {
	registry := NewRegistry()
	egRegexp := regexp.MustCompile(`.*`)

	testCases := []struct {
		name     string
		pattern  *PatternCommand
		expected bool
	}{
		{
			name: "nil pattern",
		},
		{
			name:    "empty name",
			pattern: &PatternCommand{},
		},
		{
			name: "nil handler",
			pattern: &PatternCommand{
				Name: "nohandler",
			},
		},
		{
			name: "nil pattern",
			pattern: &PatternCommand{
				Name:    "nohandler",
				Handler: mockPatternHandler{},
			},
		},
		{
			name: "valid handler",
			pattern: &PatternCommand{
				Name:    "valid",
				Handler: mockPatternHandler{},
				Pattern: egRegexp,
			},
			expected: true,
		},
		{
			name: "duplicate name",
			pattern: &PatternCommand{
				Name:    "valid",
				Handler: mockPatternHandler{},
				Pattern: egRegexp,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if actual := registry.AddPattern(tc.pattern); actual != tc.expected {
				t.Errorf("expected AddPattern(%v) to return %v got %v",
					tc.pattern, tc.expected, actual)
			}
		})
	}
}

func TestCommandRegistryAddReactionHandler(t *testing.T) {
	registry := NewRegistry()

	testCases := []struct {
		name     string
		handler  *ReactionCommand
		expected bool
	}{
		{
			name: "nil cmd",
		},
		{
			name:    "empty name",
			handler: &ReactionCommand{},
		},
		{
			name: "nil handler",
			handler: &ReactionCommand{
				Name: "nohandler",
			},
		},
		{
			name: "valid handler",
			handler: &ReactionCommand{
				Name:    "valid",
				Handler: mockReactionHandler{},
			},
			expected: true,
		},
		{
			name: "duplicate name",
			handler: &ReactionCommand{
				Name:    "valid",
				Handler: mockReactionHandler{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if actual := registry.AddReactionHandler(tc.handler); actual != tc.expected {
				t.Errorf("expected AddReactionHandler to return %v got %v", tc.expected, actual)
			}
		})
	}
}

//nolint:funlen
func TestGlobalRegistry(t *testing.T) {
	egRegexp := regexp.MustCompile(`.*`)

	commands := []*BasicCommand{
		{
			Name:    "one",
			Handler: mockCmdHandler{},
		},
		{
			Name:    "two",
			Handler: mockCmdHandler{},
		},
		{
			Name:    "three",
			Handler: mockCmdHandler{},
		},
	}
	patterns := []*PatternCommand{
		{
			Name:    "one",
			Handler: mockPatternHandler{},
			Pattern: egRegexp,
		},
		{
			Name:    "two",
			Handler: mockPatternHandler{},
			Pattern: egRegexp,
		},
	}
	reactionHandlers := []*ReactionCommand{
		{
			Name:    "one",
			Handler: mockReactionHandler{},
		},
		{
			Name:    "two",
			Handler: mockReactionHandler{},
		},
	}

	for _, cmd := range commands {
		if added := AddCommand(cmd); !added {
			t.Errorf("failed to add cmd %v to global registry", cmd)
		}
	}

	for _, pattern := range patterns {
		if added := AddPattern(pattern); !added {
			t.Errorf("failed to add pattern %v to global registry", pattern)
		}
	}

	for _, reactionHandler := range reactionHandlers {
		if added := AddReactionHandler(reactionHandler); !added {
			t.Errorf("failed to add reaction handler %v to global registry", reactionHandler)
		}
	}

	registeredCommands := DefaultRegistry.GetCommands()
	if !reflect.DeepEqual(registeredCommands, commands) {
		t.Errorf("expected to get commands: %v got %v", commands, registeredCommands)
	}

	registeredOneCmd := DefaultRegistry.GetCommand("one")
	if !reflect.DeepEqual(registeredOneCmd, commands[0]) {
		t.Errorf("expected to get command: %v from GetCommand() got %v",
			commands[0], registeredOneCmd)
	}

	unknownCmd := DefaultRegistry.GetCommand("whatever")
	if unknownCmd != nil {
		t.Errorf("expected to get nil command from GetCommand() for unknown cmd, got %v",
			unknownCmd)
	}

	registeredPatterns := DefaultRegistry.GetPatterns()
	if !reflect.DeepEqual(registeredPatterns, patterns) {
		t.Errorf("expected to get paterns: %v got %v", patterns, registeredPatterns)
	}

	registeredOnePattern := DefaultRegistry.GetPattern("one")
	if registeredOnePattern == nil {
		t.Errorf("expected to get pattern: %v from GetPattern() got nil",
			patterns[0])
	}

	unknownPattern := DefaultRegistry.GetPattern("whatever")
	if unknownPattern != nil {
		t.Errorf("expected to get nil pattern from GetPattern() for unknown name, got %v",
			unknownPattern)
	}

	registeredReactionHandlers := DefaultRegistry.GetReactionHandlers()
	if !reflect.DeepEqual(registeredReactionHandlers, reactionHandlers) {
		t.Errorf("expected to get reaction handlers: %v got %v",
			reactionHandlers, registeredReactionHandlers)
	}

	registeredOneReactionHandler := DefaultRegistry.GetReactionHandler("one")
	if registeredOneReactionHandler == nil {
		t.Errorf("expected to get reaction handler %v from GetReactionHandler() got nil",
			reactionHandlers[0])
	}

	unknownReactionHandler := DefaultRegistry.GetReactionHandler("whatever")
	if unknownReactionHandler != nil {
		t.Errorf("expected to get nil pattern from GetReactionHandler() for unknown name, got %v",
			unknownReactionHandler)
	}
}
