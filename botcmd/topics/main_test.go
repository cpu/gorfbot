//nolint:goerr113
package topics

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/cpu/gorfbot/botcmd"
	"github.com/cpu/gorfbot/slack"
	slack_mocks "github.com/cpu/gorfbot/slack/mocks"
	"github.com/cpu/gorfbot/storage"
	"github.com/cpu/gorfbot/storage/mocks"
	"github.com/cpu/gorfbot/storage/models"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus/hooks/test"
)

func TestRunParseErr(t *testing.T) {
	cmd := &topicsCmd{}

	expected := `topics: failed to parse "-hello bye": flag provided but not defined: -hello`

	if res, err := cmd.Run("-hello bye", botcmd.RunContext{}); err != nil {
		t.Errorf("unexpected run err: %v", err)
	} else if res.Message != expected {
		t.Errorf("exected run result %q got %q", expected, res)
	}
}

func TestRunHelpNoErr(t *testing.T) {
	cmd := &topicsCmd{}

	if _, err := cmd.Run("-help", botcmd.RunContext{}); err != nil {
		t.Errorf("unexpected run err: %v", err)
	}
}

func expectedOptions(channelID string, limit int64, asc bool) storage.GetTopicOptions {
	return storage.GetTopicOptions{
		Channel: channelID,
		FindOptions: storage.FindOptions{
			Limit:     limit,
			Asc:       asc,
			SortField: "date",
		},
	}
}

func setup(t *testing.T) (*gomock.Controller, *slack_mocks.MockClient, *mocks.MockStorage, botcmd.RunContext) {
	ctrl := gomock.NewController(t)
	mockSlack := slack_mocks.NewMockClient(ctrl)
	mockStorage := mocks.NewMockStorage(ctrl)
	ctx := botcmd.RunContext{
		Storage: mockStorage,
		Slack:   mockSlack,
		Message: &slack.Message{
			ChannelID: "C0000",
		},
	}

	return ctrl, mockSlack, mockStorage, ctx
}

const (
	mockTopicTimestampA = int64(1607050812)
	mockTopicTimestampB = int64(1607044110)
)

var (
	mockTimeA           = time.Unix(mockTopicTimestampA, 0)
	mockTimeB           = time.Unix(mockTopicTimestampB, 0)
	formattedTimestampA = botcmd.FormatTime(mockTimeA)
	formattedTimestampB = botcmd.FormatTime(mockTimeB)
)

func expectTopics(mockStorage *mocks.MockStorage, mockSlack *slack_mocks.MockClient, opts storage.GetTopicOptions) {
	mockSlackTimeA := fmt.Sprintf("%d.008800", mockTopicTimestampA)
	mockSlackTimeB := fmt.Sprintf("%d.001800", mockTopicTimestampB)

	topics := []models.Topic{
		{
			Channel: opts.Channel,
			Topic:   "happy new year",
			Date:    mockSlackTimeA,
			Creator: "U0000",
		},
		{
			Channel: opts.Channel,
			Topic:   "speak freely",
			Date:    mockSlackTimeB,
			Creator: "U0001",
		},
	}
	reverseTopics := []models.Topic{
		topics[1],
		topics[0],
	}

	// This is janky but it "works" for now
	if opts.Limit == 5 {
		mockSlack.EXPECT().UserName("U0000").Return("Gorf")
		mockSlack.EXPECT().UserName("U0001").Return("Garf")
		mockSlack.EXPECT().ParseTimestamp(mockSlackTimeA).Return(mockTimeA, nil)
		mockSlack.EXPECT().ParseTimestamp(mockSlackTimeB).Return(mockTimeB, nil)
	} else if opts.Limit == 1 && !opts.Asc {
		mockSlack.EXPECT().UserName("U0000").Return("Gorf")
		mockSlack.EXPECT().ParseTimestamp(mockSlackTimeA).Return(mockTimeA, nil)
	} else if opts.Limit == 1 && opts.Asc {
		mockSlack.EXPECT().UserName("U0001").Return("Garf")
		mockSlack.EXPECT().ParseTimestamp(mockSlackTimeB).Return(mockTimeB, nil)
	}

	upperLimit := opts.Limit
	if upperLimit > 2 {
		upperLimit = 2
	}

	if opts.Limit == 0 && !opts.Asc {
		// No limit, no asc
		mockStorage.EXPECT().GetTopics(opts).Return(topics, nil)
	} else if opts.Limit > 0 && !opts.Asc {
		// Limit, no asc
		mockStorage.EXPECT().GetTopics(opts).Return(topics[0:upperLimit], nil)
	} else if opts.Limit == 0 && opts.Asc {
		// No Limit, asc
		mockStorage.EXPECT().GetTopics(opts).Return(reverseTopics, nil)
	} else if opts.Limit > 0 && opts.Asc {
		// Limit, asc
		mockStorage.EXPECT().GetTopics(opts).Return(reverseTopics[0:upperLimit], nil)
	}
}

func TestStorageErr(t *testing.T) {
	ctrl, mockSlack, mockStorage, ctx := setup(t)
	defer ctrl.Finish()

	mockSlack.EXPECT().ConversationName("C0000").Return("general")

	opts := expectedOptions(ctx.Message.ChannelID, 5, false)
	mockStorage.EXPECT().GetTopics(opts).Return(nil, errors.New("topic storage err"))
	expectedErr := fmt.Sprintf(
		"topics: failed to get topics from storage opts: %v err: topic storage err",
		opts)

	log, _ := test.NewNullLogger()
	cmd := &topicsCmd{log: log}

	if _, err := cmd.Run("", ctx); err == nil {
		t.Errorf("expected err from mockStorage failure, got nil")
	} else if err.Error() != expectedErr {
		t.Errorf("expected err %q from got %q", expectedErr, err.Error())
	}
}

func TestSuccess(t *testing.T) {
	ctrl, mockSlack, mockStorage, ctx := setup(t)
	defer ctrl.Finish()

	opts := expectedOptions(ctx.Message.ChannelID, 5, false)
	expectTopics(mockStorage, mockSlack, opts)
	mockSlack.EXPECT().ConversationName(opts.Channel).Return("general")

	expectedResult := fmt.Sprintf(`:newspaper: :mega: 2 topics from channel *#general* :mega: :newspaper:
	:rolled_up_newspaper: %s - Topic changed by _Gorf_ to :scroll: *"happy new year"*
	:rolled_up_newspaper: %s - Topic changed by _Garf_ to :scroll: *"speak freely"*
`, formattedTimestampA, formattedTimestampB)
	log, _ := test.NewNullLogger()
	cmd := &topicsCmd{log: log}

	if result, err := cmd.Run("", ctx); err != nil {
		t.Errorf("unexpected err: %v", err)
	} else if result.Message != expectedResult {
		t.Errorf("expected %q got %q", expectedResult, result.Message)
	}
}

func TestSuccessLimit(t *testing.T) {
	ctrl, mockSlack, mockStorage, ctx := setup(t)
	defer ctrl.Finish()

	opts := expectedOptions(ctx.Message.ChannelID, 1, false)
	expectTopics(mockStorage, mockSlack, opts)
	mockSlack.EXPECT().ConversationName(opts.Channel).Return("general")

	expectedResult := fmt.Sprintf(`:newspaper: :mega: 1 topics from channel *#general* :mega: :newspaper:
	:rolled_up_newspaper: %s - Topic changed by _Gorf_ to :scroll: *"happy new year"*
`, formattedTimestampA)
	log, _ := test.NewNullLogger()
	cmd := &topicsCmd{log: log}

	if result, err := cmd.Run("-limit 1", ctx); err != nil {
		t.Errorf("unexpected err: %v", err)
	} else if result.Message != expectedResult {
		t.Errorf("expected %q got %q", expectedResult, result)
	}
}

func TestSuccessAsc(t *testing.T) {
	ctrl, mockSlack, mockStorage, ctx := setup(t)
	defer ctrl.Finish()

	opts := expectedOptions(ctx.Message.ChannelID, 5, true)
	expectTopics(mockStorage, mockSlack, opts)
	mockSlack.EXPECT().ConversationName(opts.Channel).Return("general")

	expectedResult := fmt.Sprintf(`:newspaper: :mega: 2 topics from channel *#general* :mega: :newspaper:
	:rolled_up_newspaper: %s - Topic changed by _Garf_ to :scroll: *"speak freely"*
	:rolled_up_newspaper: %s - Topic changed by _Gorf_ to :scroll: *"happy new year"*
`, formattedTimestampB, formattedTimestampA)
	log, _ := test.NewNullLogger()
	cmd := &topicsCmd{log: log}

	if result, err := cmd.Run("-asc", ctx); err != nil {
		t.Errorf("unexpected err: %v", err)
	} else if result.Message != expectedResult {
		t.Errorf("expected %q got %q", expectedResult, result.Message)
	}
}

func TestSuccessAscLimit(t *testing.T) {
	ctrl, mockSlack, mockStorage, ctx := setup(t)
	defer ctrl.Finish()

	opts := expectedOptions(ctx.Message.ChannelID, 1, true)
	expectTopics(mockStorage, mockSlack, opts)
	mockSlack.EXPECT().ConversationName(opts.Channel).Return("general")

	expectedResult := fmt.Sprintf(`:newspaper: :mega: 1 topics from channel *#general* :mega: :newspaper:
	:rolled_up_newspaper: %s - Topic changed by _Garf_ to :scroll: *"speak freely"*
`, formattedTimestampB)
	log, _ := test.NewNullLogger()
	cmd := &topicsCmd{log: log}

	if result, err := cmd.Run("-limit 1 -asc", ctx); err != nil {
		t.Errorf("unexpected err: %v", err)
	} else if result.Message != expectedResult {
		t.Errorf("expected %q got %q", expectedResult, result)
	}
}

func TestSuccessChannel(t *testing.T) {
	ctrl, mockSlack, mockStorage, ctx := setup(t)
	defer ctrl.Finish()

	opts := expectedOptions("C099", 1, false)
	expectTopics(mockStorage, mockSlack, opts)
	mockSlack.EXPECT().ConversationID("random").Return("C099")

	expectedResult := fmt.Sprintf(`:newspaper: :mega: 1 topics from channel *#random* :mega: :newspaper:
	:rolled_up_newspaper: %s - Topic changed by _Gorf_ to :scroll: *"happy new year"*
`, formattedTimestampA)
	log, _ := test.NewNullLogger()
	cmd := &topicsCmd{log: log}

	if result, err := cmd.Run("-limit 1 -channel random", ctx); err != nil {
		t.Errorf("unexpected err: %v", err)
	} else if result.Message != expectedResult {
		t.Errorf("expected %q got %q", expectedResult, result)
	}
}

func TestUnknownChannel(t *testing.T) {
	ctrl, mockSlack, _, ctx := setup(t)
	defer ctrl.Finish()

	mockSlack.EXPECT().ConversationID("doesnotexist").Return("")

	log, _ := test.NewNullLogger()
	cmd := &topicsCmd{log: log}
	expectedMsg := `no such channel`

	if res, err := cmd.Run("-limit 1 -channel doesnotexist", ctx); err != nil {
		t.Errorf("unexpected err: %v", err)
	} else if res.Message != expectedMsg {
		t.Errorf("expected res msg %q got %q", expectedMsg, res.Message)
	}
}

func TestConfigure(t *testing.T) {
	log, _ := test.NewNullLogger()
	cmd := &topicsCmd{}

	if err := cmd.Configure(log, nil); err != nil {
		t.Errorf("expected no err from configure")
	}

	if cmd.log != log {
		t.Errorf("expected log to be %p was %p", log, cmd.log)
	}
}
