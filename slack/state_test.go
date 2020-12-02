//nolint:goerr113
package slack

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/cpu/gorfbot/config"
	"github.com/cpu/gorfbot/test"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	real_slack "github.com/slack-go/slack"
)

func TestStateMaxAge(t *testing.T) {
	log, _ := logtest.NewNullLogger()

	// test once with explicit age configured
	maxAge := time.Minute
	testConfig := config.SlackConfig{
		StateMaxAge: &maxAge,
	}

	state := newSlackStateImpl(log, testConfig)
	if state == nil {
		t.Errorf("got unexpected nil state")
	} else if state.MaxAge() != maxAge {
		t.Errorf("expected max age %q got %q", maxAge, state.MaxAge())
	}

	// and again for default
	state = newSlackStateImpl(log, config.SlackConfig{})
	if state == nil {
		t.Errorf("got unexpected nil state")
	} else if state.MaxAge() != defaultSlackStateMaxAge {
		t.Errorf("expected max age %q got %q", maxAge, state.MaxAge())
	}
}

func TestStale(t *testing.T) {
	log, logHook := logtest.NewNullLogger()
	maxAge := time.Minute * 10
	testConfig := config.SlackConfig{
		StateMaxAge: &maxAge,
	}

	state := newSlackStateImpl(log, testConfig)
	if state == nil {
		t.Fatalf("got unexpected nil state")
	}

	expectedLogMsg := func(stale bool, lastUpdated time.Time) string {
		return fmt.Sprintf("Slack state stale? %v Last updated %s", stale, lastUpdated)
	}

	// To start with the state should always be stale
	if !state.Stale() {
		t.Errorf("expected new state to be stale")
	}

	test.ExpectLastLog(t, logHook, logrus.InfoLevel, expectedLogMsg(true, time.Time{}))
	logHook.Reset()

	// If the last update is within the max age it shouldn't be stale
	lastUpdate := time.Now().Add(time.Minute * -5)
	state.lastUpdated = lastUpdate

	if state.Stale() {
		t.Errorf("expected state to not be stale")
	}

	test.ExpectLastLog(t, logHook, logrus.InfoLevel, expectedLogMsg(false, lastUpdate))
	logHook.Reset()

	// If the last update is beyond the max age it should be stale again
	lastUpdate = time.Now().Add((maxAge + time.Minute) * -1)
	state.lastUpdated = lastUpdate

	if !state.Stale() {
		t.Errorf("expected state to be stale")
	}

	test.ExpectLastLog(t, logHook, logrus.InfoLevel, expectedLogMsg(true, lastUpdate))
	logHook.Reset()
}

func TestConversation(t *testing.T) {
	log, _ := logtest.NewNullLogger()
	testConfig := config.SlackConfig{}

	state := newSlackStateImpl(log, testConfig)
	if state == nil {
		t.Fatalf("got unexpected nil state")
	}

	egConvo := Conversation{
		ID:   "C0000",
		Name: "Fake",
	}
	state.conversationsByID[egConvo.ID] = egConvo

	if convo, found := state.Conversation(egConvo.ID); !found {
		t.Errorf("Expected to find Conversation")
	} else if convo.Name != egConvo.Name {
		t.Errorf("Expected to find name %q got %q", egConvo.Name, convo.Name)
	}
}

func TestConversationName(t *testing.T) {
	log, _ := logtest.NewNullLogger()
	testConfig := config.SlackConfig{}

	state := newSlackStateImpl(log, testConfig)
	if state == nil {
		t.Fatalf("got unexpected nil state")
	}

	egConvo := Conversation{
		ID:   "C0000",
		Name: "Fake",
	}
	state.conversationsByName[egConvo.Name] = egConvo

	if c, found := state.ConversationID(egConvo.Name); !found {
		t.Errorf("Expected to find Convo")
	} else if c.ID != egConvo.ID {
		t.Errorf("Expected to find Convo with ID %q got %q", egConvo.ID, c.ID)
	}
}

func TestUser(t *testing.T) {
	log, _ := logtest.NewNullLogger()
	testConfig := config.SlackConfig{}

	state := newSlackStateImpl(log, testConfig)
	if state == nil {
		t.Fatalf("got unexpected nil state")
	}

	egUser := User{
		ID:   "U0000",
		Name: "Gorfbot",
	}
	state.usersByID[egUser.ID] = egUser

	if u, found := state.User(egUser.ID); !found {
		t.Errorf("Expected to find User")
	} else if u.Name != egUser.Name {
		t.Errorf("Expected to find name %q got %q", egUser.Name, u.Name)
	}
}

func TestUserID(t *testing.T) {
	log, _ := logtest.NewNullLogger()
	testConfig := config.SlackConfig{}

	state := newSlackStateImpl(log, testConfig)
	if state == nil {
		t.Fatalf("got unexpected nil state")
	}

	egUser := User{
		ID:   "U0000",
		Name: "Gorfbot",
	}
	state.usersByName[egUser.Name] = egUser

	if u, found := state.UserID(egUser.Name); !found {
		t.Errorf("Expected to find User")
	} else if u.ID != egUser.ID {
		t.Errorf("Expected to find User with ID %q got %q", egUser.ID, u.ID)
	}
}

var (
	mockChanList = []real_slack.Channel{
		{
			GroupConversation: real_slack.GroupConversation{
				Conversation: real_slack.Conversation{
					ID: "C002",
				},
				Name: "general",
			},
		},
		{
			GroupConversation: real_slack.GroupConversation{
				Conversation: real_slack.Conversation{
					ID: "C001",
				},
				Name: "random",
			},
		},
	}
	mockUsersList = []real_slack.User{
		{
			ID:   "U000",
			Name: "Gorfbot",
		},
		{
			ID:   "U001",
			Name: "Garfbot",
		},
	}
)

func expectUsers(t *testing.T, state slackState) {
	for _, u := range mockUsersList {
		uID := u.ID
		expectedName := u.Name

		if user, found := state.User(uID); !found {
			t.Errorf("expected to find user %q but didn't", uID)
		} else if user.Name != expectedName {
			t.Errorf("expected user %q to have name %q but had %q",
				uID, expectedName, user.Name)
		}
	}
}

func expectConversations(t *testing.T, state slackState) {
	for _, c := range mockChanList {
		cID := c.GroupConversation.Conversation.ID
		expectedName := c.GroupConversation.Name

		if conv, found := state.Conversation(cID); !found {
			t.Errorf("expected to find conversation %q but didn't", cID)
		} else if conv.Name != expectedName {
			t.Errorf("expected conversation %q to have name %q but had %q",
				cID, expectedName, conv.Name)
		}
	}
}

func setupRefresh(
	t *testing.T, expectUsersRefresh bool, expectConversationsRefresh bool, userErr error, conversationsErr error) (
	*logrus.Logger, config.SlackConfig, *gomock.Controller, SlackAPI) {
	log, _ := logtest.NewNullLogger()
	testConfig := config.SlackConfig{}
	ctrl := gomock.NewController(t)
	mockAPI := NewMockSlackAPI(ctrl)

	if expectUsersRefresh {
		mockAPI.EXPECT().GetUsers().Return(mockUsersList, userErr)
	}

	if expectConversationsRefresh {
		getOps := &real_slack.GetConversationsParameters{
			ExcludeArchived: "true",
			Cursor:          "",
		}
		mockAPI.EXPECT().GetConversations(getOps).Return(mockChanList, "", conversationsErr)
	}

	return log, testConfig, ctrl, mockAPI
}

func TestRefreshNotStaleNoForce(t *testing.T) {
	log, testConfig, ctrl, mockAPI := setupRefresh(t, false, false, nil, nil)
	defer ctrl.Finish()

	state := newSlackStateImpl(log, testConfig)
	if state == nil {
		t.Fatalf("got unexpected nil state")
	}

	state.lastUpdated = time.Now() // pretend it's not stale yet so we can test force.

	// Calling refresh without force no calls should have been made to the mock
	if err := state.Refresh(mockAPI, false); err != nil {
		t.Errorf("unexpected refresh error: %v", err)
	}
}

func TestRefreshNotStaleForce(t *testing.T) {
	log, testConfig, ctrl, mockAPI := setupRefresh(t, true, true, nil, nil)
	defer ctrl.Finish()

	state := newSlackStateImpl(log, testConfig)
	if state == nil {
		t.Fatalf("got unexpected nil state")
	}

	state.lastUpdated = time.Now() // pretend it's not stale yet so we can test force.

	// Calling refresh with force should result in the expected mock calls
	if err := state.Refresh(mockAPI, true); err != nil {
		t.Errorf("unexpected err from Refresh")
	}
	// Users and conversations should be present
	expectUsers(t, state)
	expectConversations(t, state)
}

func TestRefreshStale(t *testing.T) {
	log, testConfig, ctrl, mockAPI := setupRefresh(t, true, true, nil, nil)
	defer ctrl.Finish()

	state := newSlackStateImpl(log, testConfig)
	if state == nil {
		t.Errorf("got unexpected nil state")
	}

	// Calling refresh with fresh state and no force should result in the expected
	// mock calls
	if err := state.Refresh(mockAPI, false); err != nil {
		t.Errorf("unexpected err from Refresh")
	}
	// Users and conversations should be present
	expectUsers(t, state)
	expectConversations(t, state)
}

func TestRefreshUsersErr(t *testing.T) {
	log, testConfig, ctrl, mockAPI := setupRefresh(
		// NB: Using a non-nil users err with setupRefresh
		t, true, true, errors.New("users are all gone man"), nil)
	defer ctrl.Finish()

	state := newSlackStateImpl(log, testConfig)
	if state == nil {
		t.Errorf("got unexpected nil state")
	}

	expectedErrMsg := `slack client failed to get users: users are all gone man`

	if err := state.Refresh(mockAPI, false); err == nil {
		t.Errorf("expected err %q, got nil", expectedErrMsg)
	} else if actual := err.Error(); actual != expectedErrMsg {
		t.Errorf("expected err %q got %q", expectedErrMsg, actual)
	}
}

func TestRefreshConversationsErr(t *testing.T) {
	log, testConfig, ctrl, mockAPI := setupRefresh(
		// NB: Using a non-nil conversations err with setupRefresh
		t, false, true, nil, errors.New("conversation is overrated"))
	defer ctrl.Finish()

	state := newSlackStateImpl(log, testConfig)
	if state == nil {
		t.Errorf("got unexpected nil state")
	}

	expectedErrMsg := `slack client failed to get conversations: slack getConversationsBatch err: conversation is overrated` //nolint:lll

	if err := state.Refresh(mockAPI, false); err == nil {
		t.Errorf("expected err %q, got nil", expectedErrMsg)
	} else if actual := err.Error(); actual != expectedErrMsg {
		t.Errorf("expected err %q got %q", expectedErrMsg, actual)
	}
}
