//nolint:goerr113,dupl
package emoji

import (
	"errors"
	"fmt"
	"testing"

	"github.com/cpu/gorfbot/botcmd"
	"github.com/cpu/gorfbot/slack"
	slack_mocks "github.com/cpu/gorfbot/slack/mocks"
	"github.com/cpu/gorfbot/storage"
	"github.com/cpu/gorfbot/storage/mocks"
	"github.com/cpu/gorfbot/storage/models"
	"github.com/golang/mock/gomock"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

const (
	fakeUserIDA = "U001"
	fakeUserIDB = "U002"
)

func TestRunParseErr(t *testing.T) {
	cmd := &emojiCmd{}
	expected := `emoji: failed to parse "-hello bye": flag provided but not defined: -hello`

	if res, err := cmd.Run("-hello bye", botcmd.RunContext{}); err != nil {
		t.Errorf("unexpected run err: %v", err)
	} else if res.Message != expected {
		t.Errorf("exected run result %q got %q", expected, res)
	}
}

func TestRunHelpNoErr(t *testing.T) {
	cmd := &emojiCmd{}
	if _, err := cmd.Run("-help", botcmd.RunContext{}); err != nil {
		t.Errorf("unexpected run err: %v", err)
	}
}

func setup() (*emojiCmd, botcmd.RunContext) {
	log, _ := logtest.NewNullLogger()
	cmd := &emojiCmd{
		log: log,
	}
	ctx := botcmd.RunContext{
		Message: &slack.Message{},
	}

	return cmd, ctx
}

func TestRunStorageErr(t *testing.T) {
	cmd, ctx := setup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	mockClient := slack_mocks.NewMockClient(ctrl)
	ctx.Storage = mockStorage
	ctx.Slack = mockClient

	ctx.Message.UserID = fakeUserIDA

	expectOpts := storage.GetEmojiOptions{
		FindOptions: storage.FindOptions{
			SortField: "count",
			Limit:     5,
		},
		User: ctx.Message.UserID,
	}

	mockClient.EXPECT().UserName(ctx.Message.UserID).Return("Gorfbot")
	mockStorage.EXPECT().GetEmoji(expectOpts).Return(nil, errors.New("data is dead"))
	expectedErr := fmt.Sprintf(
		`emoji: failed to get emoji from storage opts: %v err: data is dead`,
		expectOpts)

	if _, err := cmd.Run("", ctx); err == nil {
		t.Errorf("expected err from Run with storage err, got nil")
	} else if err.Error() != expectedErr {
		t.Errorf("expected err %q from Run, got %q", expectedErr, err.Error())
	}
}

func makeEmoji(user string, count int) []models.Emoji {
	var results []models.Emoji

	for i := 0; i < count; i++ {
		newEmoji := models.Emoji{
			User:  user,
			Emoji: ":fake:",
			Count: i + 10,
		}
		results = append(results, newEmoji)
	}

	return results
}

func TestRunNoOpts(t *testing.T) {
	cmd, ctx := setup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	mockClient := slack_mocks.NewMockClient(ctrl)
	ctx.Storage = mockStorage
	ctx.Slack = mockClient

	ctx.Message.UserID = fakeUserIDA

	expectOpts := storage.GetEmojiOptions{
		FindOptions: storage.FindOptions{
			SortField: "count",
			Limit:     5,
		},
		User: ctx.Message.UserID,
	}

	emojis := makeEmoji(ctx.Message.UserID, 5)
	mockStorage.EXPECT().GetEmoji(expectOpts).Return(emojis, nil)
	mockClient.EXPECT().UserName(ctx.Message.UserID).Return("Gorfbot")

	expectedMessage := `:upside_down_face: Top 5 observed emoji for *Gorfbot*:
	:fake: - used _10 times_.
	:fake: - used _11 times_.
	:fake: - used _12 times_.
	:fake: - used _13 times_.
	:fake: - used _14 times_.
`

	if res, err := cmd.Run("", ctx); err != nil {
		t.Errorf("unexpected err from Run with storage err, got nil")
	} else if res.Message != expectedMessage {
		t.Errorf("expected result Message %q, got %q", expectedMessage, res.Message)
	}
}

func TestRunLimit(t *testing.T) {
	cmd, ctx := setup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	mockClient := slack_mocks.NewMockClient(ctrl)
	ctx.Storage = mockStorage
	ctx.Slack = mockClient

	ctx.Message.UserID = fakeUserIDA

	expectOpts := storage.GetEmojiOptions{
		FindOptions: storage.FindOptions{
			SortField: "count",
			Limit:     2,
		},
		User: ctx.Message.UserID,
	}

	emojis := makeEmoji(ctx.Message.UserID, 2)
	mockStorage.EXPECT().GetEmoji(expectOpts).Return(emojis, nil)
	mockClient.EXPECT().UserName(ctx.Message.UserID).Return("Gorfbot")

	expectedMessage := `:upside_down_face: Top 2 observed emoji for *Gorfbot*:
	:fake: - used _10 times_.
	:fake: - used _11 times_.
`

	if res, err := cmd.Run("-limit 2", ctx); err != nil {
		t.Errorf("unexpected err from Run with storage err, got nil")
	} else if res.Message != expectedMessage {
		t.Errorf("expected result Message %q, got %q", expectedMessage, res.Message)
	}
}

func TestRunUserAndLimt(t *testing.T) {
	cmd, ctx := setup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	mockClient := slack_mocks.NewMockClient(ctrl)
	ctx.Storage = mockStorage
	ctx.Slack = mockClient

	ctx.Message.UserID = fakeUserIDA
	otherUser := fakeUserIDB

	expectOpts := storage.GetEmojiOptions{
		FindOptions: storage.FindOptions{
			SortField: "count",
			Limit:     2,
		},
		User: otherUser,
	}

	emojis := makeEmoji(otherUser, 2)
	mockStorage.EXPECT().GetEmoji(expectOpts).Return(emojis, nil)
	mockClient.EXPECT().UserID("test").Return(otherUser)

	expectedMessage := `:upside_down_face: Top 2 observed emoji for *test*:
	:fake: - used _10 times_.
	:fake: - used _11 times_.
`

	if res, err := cmd.Run("-limit 2 -user test", ctx); err != nil {
		t.Errorf("unexpected err from Run with storage err, got nil")
	} else if res.Message != expectedMessage {
		t.Errorf("expected result Message %q, got %q", expectedMessage, res.Message)
	}
}

func TestRunUserAndLimtAsc(t *testing.T) {
	cmd, ctx := setup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	mockClient := slack_mocks.NewMockClient(ctrl)
	ctx.Storage = mockStorage
	ctx.Slack = mockClient

	ctx.Message.UserID = fakeUserIDA
	otherUser := fakeUserIDB

	expectOpts := storage.GetEmojiOptions{
		FindOptions: storage.FindOptions{
			SortField: "count",
			Limit:     2,
			Asc:       true,
		},
		User: otherUser,
	}

	emojis := makeEmoji(otherUser, 2)
	mockStorage.EXPECT().GetEmoji(expectOpts).
		Return([]models.Emoji{emojis[1], emojis[0]}, nil)
	mockClient.EXPECT().UserID("test").Return(otherUser)

	expectedMessage := `:upside_down_face: Rarest 2 observed emoji for *test*:
	:fake: - used _11 times_.
	:fake: - used _10 times_.
`

	if res, err := cmd.Run("-limit 2 -asc -user test", ctx); err != nil {
		t.Errorf("unexpected err from Run with storage err, got nil")
	} else if res.Message != expectedMessage {
		t.Errorf("expected result Message %q, got %q", expectedMessage, res.Message)
	}
}

func TestRunEmoji(t *testing.T) {
	cmd, ctx := setup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	mockClient := slack_mocks.NewMockClient(ctrl)
	ctx.Storage = mockStorage
	ctx.Slack = mockClient

	ctx.Message.UserID = fakeUserIDA

	expectOpts := storage.GetEmojiOptions{
		FindOptions: storage.FindOptions{
			SortField: "count",
			Limit:     5,
		},
		User:  ctx.Message.UserID,
		Emoji: ":test:",
	}

	emojis := []models.Emoji{
		{
			User:  ctx.Message.UserID,
			Count: 99,
			Emoji: ":test:",
		},
	}
	mockStorage.EXPECT().GetEmoji(expectOpts).Return(emojis, nil)
	mockClient.EXPECT().UserName(ctx.Message.UserID).Return("Gorfbot")

	expectedMessage := "Gorfbot has used the :test: emoji 99 times\n"

	if res, err := cmd.Run("-emoji :test:", ctx); err != nil {
		t.Errorf("unexpected err from Run with storage err, got nil")
	} else if res.Message != expectedMessage {
		t.Errorf("expected result Message %q, got %q", expectedMessage, res.Message)
	}
}

func TestRunEmojiNoResult(t *testing.T) {
	cmd, ctx := setup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	mockClient := slack_mocks.NewMockClient(ctrl)
	ctx.Storage = mockStorage
	ctx.Slack = mockClient

	ctx.Message.UserID = fakeUserIDA

	expectOpts := storage.GetEmojiOptions{
		FindOptions: storage.FindOptions{
			SortField: "count",
			Limit:     5,
		},
		User:  ctx.Message.UserID,
		Emoji: ":test:",
	}

	mockStorage.EXPECT().GetEmoji(expectOpts).Return(nil, nil)
	mockClient.EXPECT().UserName(ctx.Message.UserID).Return("Gorfbot")

	expectedMessage := "Gorfbot has not been observed using emoji \":test:\"\n"

	if res, err := cmd.Run("-emoji :test:", ctx); err != nil {
		t.Errorf("unexpected err from Run with storage err, got nil")
	} else if res.Message != expectedMessage {
		t.Errorf("expected result Message %q, got %q", expectedMessage, res.Message)
	}
}

func TestConfigure(t *testing.T) {
	log, _ := logtest.NewNullLogger()
	cmd := &emojiCmd{}

	if err := cmd.Configure(log, nil); err != nil {
		t.Errorf("expected no err from configure")
	}

	if cmd.log != log {
		t.Errorf("expected log to be %p was %p", log, cmd.log)
	}
}
