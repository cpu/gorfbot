package slack

import (
	"testing"

	"github.com/slack-go/slack"
)

func TestMessage(t *testing.T) {
	m := Message{
		ChannelID: "C000",
		UserID:    "U000",
		Text:      "Hello",
		Timestamp: "123.4",
	}
	expected := `123.4 - channel C000 user U000 said "Hello"`

	if actual := m.String(); actual != expected {
		t.Errorf("expected Message %v to have str form %q got %q",
			m, expected, actual)
	}
}

func TestClientImplString(t *testing.T) {
	client := clientImpl{}
	expected := "Client waiting for ConnectedEvent"

	if actual := client.String(); actual != expected {
		t.Errorf("expected Client %v to have str form %q got %q",
			client, expected, actual)
	}

	client = clientImpl{
		botUserDetails: &slack.UserDetails{
			ID:   "U000",
			Name: "Gorf",
		},
		botTeamDetails: &slack.Team{
			ID:   "T000",
			Name: "TeamGorf",
		},
	}
	expected = `Client connected as "Gorf" (U000) to team "TeamGorf" (T000)`

	if actual := client.String(); actual != expected {
		t.Errorf("expected Client %v to have str form %q got %q",
			client, expected, actual)
	}
}
