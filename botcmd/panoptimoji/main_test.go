//nolint:goerr113
package panoptimoji

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

func setup() (*panoptimojiPattern, botcmd.RunContext, *logtest.Hook) {
	log, logHook := logtest.NewNullLogger()
	cmd := &panoptimojiPattern{
		log: log,
	}
	ctx := botcmd.RunContext{
		Message: &slack.Message{},
	}

	return cmd, ctx, logHook
}

func TestConfigure(t *testing.T) {
	log, _ := logtest.NewNullLogger()
	cmd := &panoptimojiPattern{}

	if err := cmd.Configure(log, nil); err != nil {
		t.Errorf("expected no err from configure")
	}

	if cmd.log != log {
		t.Errorf("expected log to be %p was %p", log, cmd.log)
	}
}

func TestRunNilMessage(t *testing.T) {
	cmd, ctx, _ := setup()
	ctx.Message = nil

	// Nil message
	if _, err := cmd.Run([][]string{}, ctx); err == nil {
		t.Errorf("expected err from Run w/ nil message, got nil")
	}
}

func TestRunTooFewMatches(t *testing.T) {
	cmd, ctx, _ := setup()

	// Too few submatches
	if _, err := cmd.Run([][]string{}, ctx); err == nil {
		t.Errorf("expected err from Run with empty matches, got nil")
	}
}

func TestRunTooSmallMatch(t *testing.T) {
	cmd, ctx, _ := setup()

	// Too small submatch
	if _, err := cmd.Run([][]string{{"a"}}, ctx); err == nil {
		t.Errorf("expected err from Run with empty matches, got nil")
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

	ctx.Message.UserID = "U001"

	expectEmoji := models.Emoji{
		User:  ctx.Message.UserID,
		Emoji: ":fake:",
		Count: 1,
	}

	mockStorage.EXPECT().UpsertEmojiCount(expectEmoji, false).
		Return(models.Emoji{}, errors.New("blorp failure"))

	expectedErr := `emoji usage storage returned err: blorp failure`

	if _, err := cmd.Run([][]string{{":fake:", ":fake:"}}, ctx); err == nil {
		t.Errorf("expected err from upsert with storage err, got nil")
	} else if err.Error() != expectedErr {
		t.Errorf("expected err %q from upsert, got %q", expectedErr, err.Error())
	}
}

func TestRunSuccess(t *testing.T) {
	cmd, ctx, logHook := setup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	mockClient := slack_mocks.NewMockClient(ctrl)
	ctx.Storage = mockStorage
	ctx.Slack = mockClient

	ctx.Message.UserID = "U001"

	expectEmoji := models.Emoji{
		User:  ctx.Message.UserID,
		Emoji: ":fake:",
		Count: 1,
	}
	updatedEmoji := models.Emoji{
		User:  ctx.Message.UserID,
		Emoji: ":fake:",
		Count: 1,
	}

	mockStorage.EXPECT().UpsertEmojiCount(expectEmoji, false).Return(updatedEmoji, nil)
	mockClient.EXPECT().UserName(ctx.Message.UserID).Return("Gorfbot")

	if _, err := cmd.Run([][]string{{":fake:", ":fake:"}}, ctx); err != nil {
		t.Errorf("unexpected err from run: %v\n", err)
	}

	expectedLog := `emoji usage update - User "Gorfbot" (U001) has used emoji ":fake:" (history: 2 times)`
	test.ExpectLastLog(t, logHook, logrus.InfoLevel, expectedLog)
}
