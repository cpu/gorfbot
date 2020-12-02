//nolint:goerr113,funlen
package reactjiupdate

import (
	"errors"
	"testing"

	"github.com/cpu/gorfbot/botcmd"
	"github.com/cpu/gorfbot/slack"
	slack_mocks "github.com/cpu/gorfbot/slack/mocks"
	"github.com/cpu/gorfbot/storage/mocks"
	"github.com/cpu/gorfbot/storage/models"
	"github.com/cpu/gorfbot/test"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

func setup() (*reactjiHandler, botcmd.RunContext, *logtest.Hook) {
	log, logHook := logtest.NewNullLogger()
	cmd := &reactjiHandler{
		log: log,
	}
	ctx := botcmd.RunContext{}

	return cmd, ctx, logHook
}

func TestConfigure(t *testing.T) {
	log, _ := logtest.NewNullLogger()
	cmd := &reactjiHandler{}

	if err := cmd.Configure(log, nil); err != nil {
		t.Errorf("expected no err from configure")
	}

	if cmd.log != log {
		t.Errorf("expected log to be %p was %p", log, cmd.log)
	}
}

func TestRunEmptyReaction(t *testing.T) {
	cmd, ctx, _ := setup()

	if err := cmd.Run(&slack.Reaction{}, ctx); err == nil {
		t.Errorf("expected err from Run w/ empty reaction, got nil")
	}
}

func TestRunUpsertErr(t *testing.T) {
	cmd, ctx, _ := setup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	mockClient := slack_mocks.NewMockClient(ctrl)
	ctx.Storage = mockStorage
	ctx.Slack = mockClient

	expectEmoji := models.Emoji{
		User:     "U001",
		Emoji:    ":fake:",
		Count:    1,
		Reaction: true,
	}

	reaction := &slack.Reaction{
		User:     expectEmoji.User,
		Reaction: expectEmoji.Emoji,
	}

	mockStorage.EXPECT().UpsertEmojiCount(expectEmoji, false).
		Return(models.Emoji{}, errors.New("blorp failure"))

	expectedErr := `reactji usage storage returned err: blorp failure`

	if err := cmd.Run(reaction, ctx); err == nil {
		t.Errorf("expected err from upsert with storage err, got nil")
	} else if err.Error() != expectedErr {
		t.Errorf("expected err %q from upsert, got %q", expectedErr, err.Error())
	}
}

func TestRunSuccessIncrement(t *testing.T) {
	cmd, ctx, logHook := setup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	mockClient := slack_mocks.NewMockClient(ctrl)
	ctx.Storage = mockStorage
	ctx.Slack = mockClient

	expectEmoji := models.Emoji{
		User:     "U001",
		Emoji:    ":fake:",
		Count:    1,
		Reaction: true,
	}

	reaction := &slack.Reaction{
		User:     expectEmoji.User,
		Reaction: expectEmoji.Emoji,
	}
	updatedEmoji := models.Emoji{
		User:  expectEmoji.User,
		Emoji: expectEmoji.Emoji,
		Count: 2,
	}

	mockStorage.EXPECT().UpsertEmojiCount(expectEmoji, false).Return(updatedEmoji, nil)
	mockClient.EXPECT().UserName(expectEmoji.User).Return("Gorfbot")

	if err := cmd.Run(reaction, ctx); err != nil {
		t.Errorf("unexpected err from run: %v\n", err)
	}

	expectedLog := `reactji usage update - User "Gorfbot" (U001) reacted with reactji ":fake:" (history: 3 times)`
	test.ExpectLastLog(t, logHook, logrus.InfoLevel, expectedLog)
}

func TestRunSuccessDecrement(t *testing.T) {
	cmd, ctx, logHook := setup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	mockClient := slack_mocks.NewMockClient(ctrl)
	ctx.Storage = mockStorage
	ctx.Slack = mockClient

	expectEmoji := models.Emoji{
		User:     "U001",
		Emoji:    ":fake:",
		Count:    0,
		Reaction: true,
	}

	reaction := &slack.Reaction{
		User:     expectEmoji.User,
		Reaction: expectEmoji.Emoji,
		Removed:  true,
	}
	updatedEmoji := models.Emoji{
		User:  expectEmoji.User,
		Emoji: expectEmoji.Emoji,
		Count: 1,
	}

	mockStorage.EXPECT().UpsertEmojiCount(expectEmoji, true).Return(updatedEmoji, nil)
	mockClient.EXPECT().UserName(expectEmoji.User).Return("Gorfbot")

	if err := cmd.Run(reaction, ctx); err != nil {
		t.Errorf("unexpected err from run: %v\n", err)
	}

	expectedLog := `reactji usage update - User "Gorfbot" (U001) removed reactji ":fake:" (new history: 0 times)`
	test.ExpectLastLog(t, logHook, logrus.InfoLevel, expectedLog)
}
