package slack

import (
	"errors"
	"fmt"
	"time"

	"github.com/cpu/gorfbot/config"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

// Client is an interface that abstracts away the slack client from the rest of the
// codebase. It contains all of the operations the bot needs to be able to do with
// Slack.
//go:generate mockgen -destination=mocks/mock_client.go -package=mocks . Client
type Client interface {
	// Listen starts the client running forever and is intended to be called from
	// a dedicated goroutine - it will not return. Real time events for messages
	// and reactions are dispatched to the provided channels.
	Listen(msgChan chan<- *Message, reactionChan chan<- *Reaction)
	// SendMessage sends the provided text to the provided slack channel ID.
	SendMessage(text, channelID string)
	// AddReaction adds the provided reaction (no ":" delimiters) to the given
	// message.
	AddReaction(reaction string, message *Message) error
	// ParseTimestamp parses a Slack-style timestamp and returns a time.Time
	// instance.
	ParseTimestamp(ts string) (time.Time, error)
	// BotName returns the friendly name of the bot.
	BotName() string
	// BotID returns the user ID of the bot.
	BotID() string
	// TeamName returns the friendly name of the slack team/workspace.
	TeamName() string
	// TeamID returns the team ID the bot is connected to.
	TeamID() string
	// ConversationName translates a slack channel/conversation ID to a friendly name.
	ConversationName(id string) string
	// ConversationID is the reverse of ConversationName and returns the ID for
	// a friendly channel/conversation name.
	ConversationID(name string) string
	// UserName returns the friendly user name for the given slack user ID.
	UserName(id string) string
	// UserID is the reverse of Username and returns the ID for a friendly user
	// name.
	UserID(username string) string
}

// SlackAPI is an interface that abstracts away the slack API from this package's
// code. It is 100% to facilitate unit testing/mocking and shouldn't be used outside
// of this package. It must be exported to have mockgen work :-(
//
// NB: Generate this mock in the same package so we avoid a cyclic import - it's
// most useful in slack_test.go.
//go:generate mockgen -destination=mock_api.go -package=slack . SlackAPI
type SlackAPI interface { //nolint:golint
	GetConversations(opts *slack.GetConversationsParameters) ([]slack.Channel, string, error)
	GetUsers() ([]slack.User, error)
}

// Message is a structure describing a slack message from a user in a channel.
type Message struct {
	// ChannelID is the ID of the channel that the message was seen in.
	ChannelID string
	// UserID is the ID of the user that spoke the message.
	UserID string
	// Text is the raw text of the message.
	Text string
	// Timestamp is the raw slack timestamp of the message.
	Timestamp string
}

// String is a simple debugging representation of a message.
func (m Message) String() string {
	return fmt.Sprintf("%s - channel %s user %s said %q",
		m.Timestamp, m.ChannelID, m.UserID, m.Text)
}

// Conversation is a structure describing a channel/conversation. It has both an ID
// and a friendly name. Note: typically a Conversation is a channel but it may also
// be a DM exchange!
type Conversation struct {
	// Conversation ID.
	ID string
	// Conversation name (no "#" prefix for channel names).
	Name string
}

// User is a structure describing a slack user. It has both an ID and a friendly
// name.
type User struct {
	// User ID.
	ID string
	// User's friendly name (no leading "@" prefix).
	Name string
}

// Reaction is a structure describing a reaction event.
type Reaction struct {
	// The user ID that added/removed the reactji.
	User string
	// The reactji (no ":" delimiters).
	Reaction string
	// The slack timestamp that the reaction event occurred.
	Timestamp string
	// Whether the reactji was removed or added.
	Removed bool
}

// clientImpl is the implementation of the Client interface.
type clientImpl struct {
	log            *logrus.Logger
	config         config.SlackConfig
	rtm            *slack.RTM
	botUserDetails *slack.UserDetails
	botTeamDetails *slack.Team
	state          slackState
}

// clientLogger is a simple adapter for the slack logger interface that dumps
// all messages to Info of the logrus logger it wraps.
type clientLogger struct {
	log *logrus.Logger
}

// Output ignores the provided level int and always logs to the logrus Info
// level of the wrapped logger.
func (c clientLogger) Output(level int, msg string) error {
	c.log.Info(msg)
	return nil
}

// New Constructs Client instance from the given config or returns an error.
// After calling New a managed RTM instance for a Websocket with slack will
// have been created and spawned on a goroutine and Listen() may be called to
// read events.
func New(log *logrus.Logger, c *config.Config) (Client, error) {
	if log == nil {
		log = logrus.New()
	}

	if c == nil {
		return nil, fmt.Errorf("slack client err: %w", config.ErrNilConfig)
	}

	if err := c.SlackConf.Check(); err != nil {
		return nil, fmt.Errorf("slack client config err: %w", err)
	}

	// Create a slack client instance with the Slack client library. Use a
	// clientLogger to adapt the slack logs to logrus.
	client := slack.New(
		c.SlackConf.APIToken,
		slack.OptionDebug(c.SlackConf.Debug),
		slack.OptionLog(clientLogger{log: log}))

	// Start processing real time message events. Let the Slack client library manage
	// the connection in its own goroutine.
	rtm := client.NewRTM()
	go rtm.ManageConnection()

	clientImpl := &clientImpl{
		log:    log,
		config: c.SlackConf,
		rtm:    rtm,
		state:  newSlackStateImpl(log, c.SlackConf),
	}
	// Keep the slack state cache updated in a dedicated goroutine.
	go clientImpl.updateState()

	return clientImpl, nil
}

// updateState will update the slack state when required, sleeping for the
// configured max age minus one second.
func (c clientImpl) updateState() {
	for {
		c.log.Info("updateState goroutine waking up to try state refresh")

		if err := c.state.Refresh(c.rtm, false); err != nil {
			c.log.Errorf("error updating slack client state: %v", err)
		}

		c.log.Infof("updateState goroutine sleeping for %s", c.state.MaxAge())
		time.Sleep(c.state.MaxAge() - time.Second)
	}
}

// String is a simple debugging representation of the client's connection state.
func (c clientImpl) String() string {
	if c.botUserDetails == nil || c.botTeamDetails == nil {
		return "Client waiting for ConnectedEvent"
	}

	return fmt.Sprintf("Client connected as %q (%s) to team %q (%s)",
		c.BotName(), c.BotID(), c.TeamName(), c.TeamID())
}

// Listen begins processing Slack RTM incoming events and dispatches them as types
// from this package.
func (c *clientImpl) Listen(msgChan chan<- *Message, reactionChan chan<- *Reaction) {
	for msg := range c.rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.ConnectedEvent:
			c.log.Infof("Connected to Slack (%v)", ev.ConnectionCount)

			if ev.Info == nil {
				c.log.Warn("ConnectedEvent received with nil ev.Info")

				continue
			}

			c.botUserDetails = ev.Info.User
			c.botTeamDetails = ev.Info.Team
			c.log.Info(c)

		case *slack.MessageEvent:
			msgChan <- &Message{
				Timestamp: ev.Msg.Timestamp,
				ChannelID: ev.Msg.Channel,
				UserID:    ev.Msg.User,
				Text:      ev.Msg.Text,
			}

		case *slack.ReactionAddedEvent:
			reactionChan <- &Reaction{
				Timestamp: ev.EventTimestamp,
				User:      ev.User,
				Reaction:  ev.Reaction,
			}

		case *slack.ReactionRemovedEvent:
			reactionChan <- &Reaction{
				Timestamp: ev.EventTimestamp,
				User:      ev.User,
				Reaction:  ev.Reaction,
				Removed:   true,
			}

		case *slack.LatencyReport:
			c.log.Tracef("Current latency: %v\n", ev.Value)

		case *slack.RTMError:
			c.log.Errorf("Slack RTM Error: %v\n", ev)

		case *slack.InvalidAuthEvent:
			c.log.Errorf("Invalid credentials\n")
			return
		}
	}
}

func (c clientImpl) SendMessage(text, channelID string) {
	c.rtm.SendMessage(c.rtm.NewOutgoingMessage(text, channelID))
}

var errNilMessage = errors.New("add reaction failed: message is nil")

func (c clientImpl) AddReaction(reaction string, message *Message) error {
	if message == nil {
		return errNilMessage
	}

	item := slack.ItemRef{
		Channel:   message.ChannelID,
		Timestamp: message.Timestamp,
	}

	return c.rtm.AddReaction(reaction, item)
}

func (c clientImpl) BotName() string {
	if c.botUserDetails != nil {
		return c.botUserDetails.Name
	}

	return ""
}

func (c clientImpl) BotID() string {
	if c.botUserDetails != nil {
		return c.botUserDetails.ID
	}

	return ""
}

func (c clientImpl) TeamName() string {
	if c.botTeamDetails != nil {
		return c.botTeamDetails.Name
	}

	return ""
}

func (c clientImpl) TeamID() string {
	if c.botTeamDetails != nil {
		return c.botTeamDetails.ID
	}

	return ""
}

func (c clientImpl) ConversationName(id string) string {
	if conversation, found := c.state.Conversation(id); found {
		return conversation.Name
	}

	return ""
}

func (c clientImpl) ConversationID(name string) string {
	if conversation, found := c.state.ConversationID(name); found {
		return conversation.ID
	}

	return ""
}

func (c clientImpl) UserName(id string) string {
	if user, found := c.state.User(id); found {
		return user.Name
	}

	return ""
}

func (c clientImpl) UserID(username string) string {
	if user, found := c.state.UserID(username); found {
		return user.ID
	}

	return ""
}
