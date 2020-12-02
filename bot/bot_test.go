//nolint:goerr113,funlen
package bot

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/cpu/gorfbot/botcmd"
	"github.com/cpu/gorfbot/botcmd/mocks"
	"github.com/cpu/gorfbot/slack"
	slack_mocks "github.com/cpu/gorfbot/slack/mocks"
	"github.com/cpu/gorfbot/test"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

func TestTryMessageAsPattern(t *testing.T) {
	log, logHook := logtest.NewNullLogger()

	testCases := []struct {
		name                string
		message             *slack.Message
		handlerErr          error
		handlerResponse     botcmd.RunResult
		expectHandlerCalled bool
		expectedMatches     [][]string
		expectClientCalled  bool
	}{
		{
			name:    "no match",
			message: &slack.Message{Text: "whatever"},
		},
		{
			name:                "pattern match, no response",
			message:             &slack.Message{Text: "hello world"},
			handlerResponse:     botcmd.RunResult{},
			expectHandlerCalled: true,
			expectedMatches:     [][]string{{"hello world", "world"}},
		},
		{
			name:                "pattern match, with response",
			message:             &slack.Message{Text: "hello world"},
			handlerResponse:     botcmd.RunResult{Message: "hello world!!!"},
			expectClientCalled:  true,
			expectHandlerCalled: true,
			expectedMatches:     [][]string{{"hello world", "world"}},
		},
		{
			name:                "pattern match, err",
			message:             &slack.Message{Text: "hello world"},
			handlerErr:          errors.New("danger danger"),
			expectHandlerCalled: true,
			expectedMatches:     [][]string{{"hello world", "world"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer logHook.Reset()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create a mock command handler and register it with the registry used for the
			// bot.
			mockHandler := mocks.NewMockPatternHandler(ctrl)
			cmdRegistry := botcmd.NewRegistry()
			cmdRegistry.AddPattern(&botcmd.PatternCommand{
				Name:    "test",
				Handler: mockHandler,
				Pattern: regexp.MustCompile("hello (.*)"),
			})

			// Create a mock Slack client.
			mockClient := slack_mocks.NewMockClient(ctrl)

			// Create a bot with the mock logger, test registry, and mock slack client.
			bot := botImpl{
				log:      log,
				registry: cmdRegistry,
				slack:    mockClient,
			}

			// If we expected the client was called with a response, set that up with
			// the mock
			if tc.expectClientCalled {
				mockClient.EXPECT().SendMessage(tc.handlerResponse.Message, tc.message.ChannelID)
			}

			// If we expected the handler is called, set that up with the mock
			if tc.expectHandlerCalled {
				mockHandler.EXPECT().
					Run(tc.expectedMatches, botcmd.RunContext{
						Message: tc.message,
						Slack:   mockClient,
					}).
					Return(tc.handlerResponse, tc.handlerErr)
			}

			// Handle the message
			bot.tryMessageAsPattern(tc.message)

			if tc.handlerErr != nil {
				expectedLog := `Pattern "test" returned an error: danger danger`
				test.ExpectLastLog(t, logHook, logrus.ErrorLevel, expectedLog)
			}
		})
	}
}

func TestTryMessageAsCommand(t *testing.T) {
	testCases := []struct {
		name                string
		message             *slack.Message
		expectHandlerCalled bool
		expectBotIDCalled   bool
		expectedRest        string
	}{
		{
			name:    "no prefix/mention",
			message: &slack.Message{Text: ""},
		},
		{
			name:                "bare command, no rest",
			message:             &slack.Message{Text: "!test"},
			expectHandlerCalled: true,
			expectedRest:        "",
		},
		{
			name:                "bare command, with rest",
			message:             &slack.Message{Text: "!test hello world!!!"},
			expectHandlerCalled: true,
			expectedRest:        "hello world!!!",
		},
		{
			name:              "mention, wrong user",
			message:           &slack.Message{Text: "<@UXXX>"},
			expectBotIDCalled: true,
		},
		{
			name:              "mention, bot user, too short",
			message:           &slack.Message{Text: "<@U000>"},
			expectBotIDCalled: true,
		},
		{
			name:                "mention, bot user, prefix, no rest",
			message:             &slack.Message{Text: "<@U000> !test"},
			expectBotIDCalled:   true,
			expectHandlerCalled: true,
			expectedRest:        "",
		},
		{
			name:                "mention, bot user, prefix, rest",
			message:             &slack.Message{Text: "<@U000> !test hello world!!!"},
			expectBotIDCalled:   true,
			expectHandlerCalled: true,
			expectedRest:        "hello world!!!",
		},
		{
			name:                "mention, bot user, no prefix, no rest",
			message:             &slack.Message{Text: "<@U000> test"},
			expectBotIDCalled:   true,
			expectHandlerCalled: true,
			expectedRest:        "",
		},
		{
			name:                "mention, bot user, no prefix, rest",
			message:             &slack.Message{Text: "<@U000> test hello world!!!"},
			expectBotIDCalled:   true,
			expectHandlerCalled: true,
			expectedRest:        "hello world!!!",
		},
	}

	log, logHook := logtest.NewNullLogger()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer logHook.Reset()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create a mock Slack client.
			mockClient := slack_mocks.NewMockClient(ctrl)

			// Create a mock command handler and register it with the registry used for the
			// bot.
			mockHandler := mocks.NewMockCommandHandler(ctrl)
			cmdRegistry := botcmd.NewRegistry()
			cmdRegistry.AddCommand(&botcmd.BasicCommand{
				Name:    "test",
				Handler: mockHandler,
			})

			// Create a bot with the mock logger, test registry and mock slack client.
			bot := botImpl{
				log:      log,
				registry: cmdRegistry,
				slack:    mockClient,
			}

			// If we expected the bot ID was looked up, set that up with the mock
			if tc.expectBotIDCalled {
				mockClient.EXPECT().BotID().Return("U000")
			}

			// If we expected the handler is called, set that up with the mock
			if tc.expectHandlerCalled {
				mockHandler.EXPECT().Run(tc.expectedRest, botcmd.RunContext{
					Message: tc.message,
					Slack:   mockClient,
				}).Return(botcmd.RunResult{}, nil)
			}

			// Handle the message
			bot.tryMessageAsCommand(tc.message)
		})
	}
}

func TestHandleCommandMessageEmptyCmd(t *testing.T) {
	log, logHook := logtest.NewNullLogger()
	defer logHook.Reset()

	bot := botImpl{
		log:      log,
		registry: botcmd.NewRegistry(),
	}

	// An empty cmd should warn
	bot.handleCommandMessage("", "", nil)

	test.ExpectLastLog(
		t, logHook, logrus.WarnLevel, "Got empty command name in handleCommandMessage")
}

func TestHandleCommandMessageUnknownCmd(t *testing.T) {
	log, logHook := logtest.NewNullLogger()
	defer logHook.Reset()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := slack_mocks.NewMockClient(ctrl)

	bot := botImpl{
		log:      log,
		registry: botcmd.NewRegistry(),
		slack:    mockClient,
	}

	mockClient.EXPECT().AddReaction("interrobang", nil).Return(nil)

	// An unknown cmd should warn
	// An empty cmd should warn
	bot.handleCommandMessage("blorp", "", nil)

	expectedLogs := []*logrus.Entry{
		{
			Level:   logrus.WarnLevel,
			Message: `Command "blorp" not registered with bot`,
		},
		{
			Level:   logrus.InfoLevel,
			Message: `Adding reactions: ["interrobang"]`,
		},
	}
	test.ExpectLogs(t, logHook, expectedLogs)
}

func TestHandleCommandMessageError(t *testing.T) {
	log, logHook := logtest.NewNullLogger()
	defer logHook.Reset()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock command handler and register it with the registry used for the
	// bot.
	mockHandler := mocks.NewMockCommandHandler(ctrl)
	cmdRegistry := botcmd.NewRegistry()
	cmdRegistry.AddCommand(&botcmd.BasicCommand{
		Name:    "test",
		Handler: mockHandler,
	})

	mockClient := slack_mocks.NewMockClient(ctrl)
	mockClient.EXPECT().AddReaction("negative_squared_cross_mark", nil)

	// Create a bot with the mock logger and test registry.
	bot := botImpl{
		log:      log,
		registry: cmdRegistry,
		slack:    mockClient,
	}
	ctx := botcmd.RunContext{
		Slack: mockClient,
	}

	// Mock an error being returned from the test command
	mockHandler.EXPECT().Run("hello", ctx).
		Return(botcmd.RunResult{}, errors.New("bogus"))

	// Then handle a command message for the test command
	bot.handleCommandMessage("test", "hello", nil)

	expectedLogs := []*logrus.Entry{
		{
			Level:   logrus.ErrorLevel,
			Message: `Command "test" returned an error: bogus`,
		},
		{
			Level:   logrus.InfoLevel,
			Message: `Adding reactions: ["negative_squared_cross_mark"]`,
		},
	}
	test.ExpectLogs(t, logHook, expectedLogs)
}

func TestHandleCommandMessageReturnEmpty(t *testing.T) {
	log, logHook := logtest.NewNullLogger()
	defer logHook.Reset()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock command handler and register it with the registry used for the
	// bot.
	mockHandler := mocks.NewMockCommandHandler(ctrl)
	cmdRegistry := botcmd.NewRegistry()
	cmdRegistry.AddCommand(&botcmd.BasicCommand{
		Name:    "test",
		Handler: mockHandler,
	})

	// Create a mock Slack client.
	mockClient := slack_mocks.NewMockClient(ctrl)

	// Create a bot with the mock logger and test registry.
	bot := botImpl{
		log:      log,
		registry: cmdRegistry,
		slack:    mockClient,
	}

	// Mock an empty, non-err response being returned from the test command
	mockHandler.EXPECT().Run("hello", botcmd.RunContext{Slack: mockClient}).
		Return(botcmd.RunResult{}, nil)

	// Then handle a command message for the test command
	bot.handleCommandMessage("test", "hello", nil)

	// There should be no log events
	if le := logHook.LastEntry(); le != nil {
		t.Errorf("unexpected log event: %v", le)
	}
}

func TestHandleCommandMessageReturnMsg(t *testing.T) {
	log, logHook := logtest.NewNullLogger()
	defer logHook.Reset()

	log.SetLevel(logrus.TraceLevel)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock command handler and register it with the registry used for the
	// bot.
	mockHandler := mocks.NewMockCommandHandler(ctrl)
	cmdRegistry := botcmd.NewRegistry()
	cmdRegistry.AddCommand(&botcmd.BasicCommand{
		Name:    "test",
		Handler: mockHandler,
	})

	// Create a mock Slack client.
	mockClient := slack_mocks.NewMockClient(ctrl)

	// Create a bot with the mock logger and test registry.
	bot := botImpl{
		log:      log,
		registry: cmdRegistry,
		slack:    mockClient,
	}

	mockMsg := &slack.Message{
		ChannelID: "C000",
	}
	respMsg := botcmd.RunResult{
		Message: "Hello World!!!",
	}

	// Mock a non-empty, non-err response being returned from the test command
	mockHandler.EXPECT().Run("hello", botcmd.RunContext{
		Message: mockMsg,
		Slack:   mockClient,
	}).Return(respMsg, nil)

	// We expect the slack client to be told to send the reply to the right channel
	mockClient.EXPECT().SendMessage(respMsg.Message, mockMsg.ChannelID)

	// Then handle a command message for the test command
	bot.handleCommandMessage("test", "hello", mockMsg)

	expectedMsg := fmt.Sprintf(`Posting returned msg %q`, respMsg.Message)
	test.ExpectLastLog(t, logHook, logrus.TraceLevel, expectedMsg)
}

func TestHandleCommandMessageReturnReactji(t *testing.T) {
	log, logHook := logtest.NewNullLogger()
	defer logHook.Reset()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock command handler and register it with the registry used for the
	// bot.
	mockHandler := mocks.NewMockCommandHandler(ctrl)
	cmdRegistry := botcmd.NewRegistry()
	cmdRegistry.AddCommand(&botcmd.BasicCommand{
		Name:    "test",
		Handler: mockHandler,
	})

	// Create a mock Slack client.
	mockClient := slack_mocks.NewMockClient(ctrl)

	// Create a bot with the mock logger and test registry.
	bot := botImpl{
		log:      log,
		registry: cmdRegistry,
		slack:    mockClient,
	}

	mockMsg := &slack.Message{
		ChannelID: "C000",
		Timestamp: "1111",
	}
	respMsg := botcmd.RunResult{
		Reactji: []string{"thumbsup", "thumbsdown"},
	}

	// Mock a non-empty, non-err response being returned from the test command
	mockHandler.EXPECT().Run("hello", botcmd.RunContext{
		Message: mockMsg,
		Slack:   mockClient,
	}).Return(respMsg, nil)

	// We expect the slack client to be told to add reactji
	mockClient.EXPECT().AddReaction("thumbsup", mockMsg).Return(nil)
	mockClient.EXPECT().AddReaction("thumbsdown", mockMsg).Return(nil)

	// Then handle a command message for the test command
	bot.handleCommandMessage("test", "hello", mockMsg)

	expectedMsg := `Adding reactions: ["thumbsup" "thumbsdown"]`
	test.ExpectLastLog(t, logHook, logrus.InfoLevel, expectedMsg)
}

func TestAddReactionsErr(t *testing.T) {
	log, logHook := logtest.NewNullLogger()
	defer logHook.Reset()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := slack_mocks.NewMockClient(ctrl)
	mockClient.EXPECT().AddReaction("whatever", nil).Return(errors.New("bogus"))

	bot := botImpl{
		log:   log,
		slack: mockClient,
	}

	bot.addReactions([]string{"whatever"}, nil)
	test.ExpectLastLog(t, logHook, logrus.ErrorLevel, `Failed to add reaction: bogus`)
}
