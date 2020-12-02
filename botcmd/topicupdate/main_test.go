//nolint:goerr113
package topicupdate

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cpu/gorfbot/botcmd"
	"github.com/cpu/gorfbot/slack"
	"github.com/cpu/gorfbot/storage/mocks"
	"github.com/cpu/gorfbot/storage/models"
	"github.com/cpu/gorfbot/test"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

func setup() (*topicUpdatePattern, botcmd.RunContext, *logtest.Hook) {
	log, logHook := logtest.NewNullLogger()
	cmd := &topicUpdatePattern{
		log: log,
	}
	ctx := botcmd.RunContext{}

	return cmd, ctx, logHook
}

func TestRunNilMessage(t *testing.T) {
	cmd, ctx, _ := setup()

	// Nil message
	if _, err := cmd.Run([][]string{}, ctx); err == nil {
		t.Errorf("expected err from Run w/ nil message, got nil")
	}
}

func TestRunTooManySubMatches(t *testing.T) {
	cmd, ctx, _ := setup()

	ctx.Message = &slack.Message{
		ChannelID: "C000",
		Timestamp: "33333",
	}

	// Too few submatches
	if _, err := cmd.Run([][]string{{"a", "b"}, {"c", "d"}}, ctx); err == nil {
		t.Errorf("expected err from run with too many submatches got nil")
	}
}

func TestRunTooFewMatches(t *testing.T) {
	cmd, ctx, _ := setup()

	ctx.Message = &slack.Message{
		ChannelID: "C000",
		Timestamp: "33333",
	}

	// Too few submatches
	if _, err := cmd.Run([][]string{{"a", "b"}}, ctx); err == nil {
		t.Errorf("expected err from Run('', nil) got nil")
	}
}

func TestStorageError(t *testing.T) {
	cmd, ctx, _ := setup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	ctx.Storage = mockStorage
	ctx.Message = &slack.Message{
		ChannelID: "C000",
		Timestamp: "33333",
	}
	subMatches := [][]string{{"", "U000", "abcd"}}
	expectedTopic := models.Topic{
		Creator: "U000",
		Channel: "C000",
		Topic:   "abcd",
		Date:    "33333",
	}
	mockStorage.EXPECT().
		AddTopic(expectedTopic).
		Return(errors.New("big mongus err"))

	expectedErr := `topic updates pattern error storing new topic: big mongus err`

	if _, err := cmd.Run(subMatches, ctx); err == nil {
		t.Errorf("expected err %q, got nil", expectedErr)
	} else if actual := err.Error(); actual != expectedErr {
		t.Errorf("expected err %q, got %q", expectedErr, actual)
	}
}

func TestSuccess(t *testing.T) {
	cmd, ctx, logHook := setup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	ctx.Storage = mockStorage
	ctx.Message = &slack.Message{
		ChannelID: "C000",
		Timestamp: "33333",
	}
	subMatches := [][]string{{"", "U000", "abcd"}}
	expectedTopic := models.Topic{
		Creator: "U000",
		Channel: "C000",
		Topic:   "abcd",
		Date:    "33333",
	}
	mockStorage.EXPECT().AddTopic(expectedTopic).Return(nil)

	expectedReactji := []string{"mag", "newspaper"}

	if res, err := cmd.Run(subMatches, ctx); err != nil {
		t.Errorf("unexpected err: %q", err)
	} else if !reflect.DeepEqual(res.Reactji, expectedReactji) {
		t.Errorf("expected reactji %q got %q", expectedReactji, res.Reactji)
	}

	test.ExpectLastLog(t, logHook, logrus.InfoLevel, expectedTopic.String())
}

func TestConfigure(t *testing.T) {
	log, _ := logtest.NewNullLogger()
	cmd := &topicUpdatePattern{}

	if err := cmd.Configure(log, nil); err != nil {
		t.Errorf("expected no err from configure")
	}

	if cmd.log != log {
		t.Errorf("expected log to be %p was %p", log, cmd.log)
	}
}
