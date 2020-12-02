package slack

import (
	"fmt"
	"sync"
	"time"

	"github.com/cpu/gorfbot/config"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

var (
	defaultSlackStateMaxAge = time.Hour
)

// slackState is an interface describing a cache of slack state. It's mostly to
// facilitate unit testing with mock state.
type slackState interface {
	Stale() bool
	MaxAge() time.Duration
	Refresh(client SlackAPI, force bool) error
	Conversation(id string) (Conversation, bool)
	ConversationID(name string) (Conversation, bool)
	User(id string) (User, bool)
	UserID(username string) (User, bool)
}

// slackStateImpl is a simple concurrency safe cache of slack API state. It
// periodically fetches the entire conversation and user list from the connected
// slack team. Using this information it's possible to quickly map
// conversation/user IDs to friendly names (and vice-versa). Ideally
// cache-misses would be checked against the API directly to help with situations
// where new users/channels are created but yet known by the bot. In practice for
// the very small & sleepy Slack's this bot was built for it isn't a problem.
type slackStateImpl struct {
	sync.RWMutex

	log                 *logrus.Logger
	maxAge              time.Duration
	lastUpdated         time.Time
	conversationsByID   map[string]Conversation
	conversationsByName map[string]Conversation
	usersByID           map[string]User
	usersByName         map[string]User
}

func newSlackStateImpl(log *logrus.Logger, config config.SlackConfig) *slackStateImpl {
	if log == nil {
		log = logrus.New()
	}

	maxAge := defaultSlackStateMaxAge
	if config.StateMaxAge != nil {
		maxAge = *config.StateMaxAge
	}

	return &slackStateImpl{
		log:                 log,
		maxAge:              maxAge,
		conversationsByID:   make(map[string]Conversation),
		conversationsByName: make(map[string]Conversation),
		usersByID:           make(map[string]User),
		usersByName:         make(map[string]User),
	}
}

func (s *slackStateImpl) MaxAge() time.Duration {
	s.RLock()
	defer s.RUnlock()

	return s.maxAge
}

func (s *slackStateImpl) Stale() bool {
	s.RLock()
	defer s.RUnlock()
	stale := s.lastUpdated.Add(s.maxAge).Before(time.Now())

	s.log.Infof("Slack state stale? %v Last updated %s", stale, s.lastUpdated)

	return stale
}

func (s *slackStateImpl) Conversation(id string) (Conversation, bool) {
	s.RLock()
	defer s.RUnlock()

	channel, found := s.conversationsByID[id]

	return channel, found
}

func (s *slackStateImpl) ConversationID(name string) (Conversation, bool) {
	s.RLock()
	defer s.RUnlock()

	channel, found := s.conversationsByName[name]

	return channel, found
}

func (s *slackStateImpl) User(id string) (User, bool) {
	s.RLock()
	defer s.RUnlock()

	user, found := s.usersByID[id]

	return user, found
}

func (s *slackStateImpl) UserID(username string) (User, bool) {
	s.RLock()
	defer s.RUnlock()

	user, found := s.usersByName[username]

	return user, found
}

func (s *slackStateImpl) Refresh(client SlackAPI, force bool) error {
	// Only refresh if stale or forced.
	if !s.Stale() && !force {
		return nil
	}

	// Fetch full conversation list and populate a new ID map
	newConversationsByID := make(map[string]Conversation)
	newConversationsByName := make(map[string]Conversation)

	conversations, err := s.conversations(client)
	if err != nil {
		return err
	}

	for _, conversation := range conversations {
		newConversationsByID[conversation.ID] = conversation
		newConversationsByName[conversation.Name] = conversation
	}

	// Fetch full user list and populate a new ID map
	newUsersByID := make(map[string]User)
	newUsersByName := make(map[string]User)

	users, err := s.users(client)
	if err != nil {
		return err
	}

	for _, user := range users {
		newUsersByID[user.ID] = user
		newUsersByName[user.Name] = user
	}

	// Lock for update, replace maps
	s.Lock()
	defer s.Unlock()
	s.conversationsByID = newConversationsByID
	s.conversationsByName = newConversationsByName
	s.usersByID = newUsersByID
	s.usersByName = newUsersByName

	return nil
}

func (s *slackStateImpl) users(client SlackAPI) ([]User, error) {
	s.log.Infof("Fetching users")

	users, err := client.GetUsers()
	if err != nil {
		return nil, fmt.Errorf("slack client failed to get users: %w", err)
	}

	s.log.Infof("Found %d users", len(users))

	results := make([]User, len(users))
	for i, user := range users {
		results[i] = User{
			ID:   user.ID,
			Name: user.Name,
		}
	}

	return results, nil
}

func (s *slackStateImpl) conversations(client SlackAPI) ([]Conversation, error) {
	var apiResults []slack.Channel

	s.log.Infof("Fetching batch of conversations")

	firstBatch, cursor, err := getConversationsBatch(client, "")
	if err != nil {
		return nil, fmt.Errorf("slack client failed to get conversations: %w", err)
	}

	apiResults = append(apiResults, firstBatch...)
	s.log.Infof("Found %d conversations so far. Next cursor: %q", len(apiResults), cursor)

	batchNumber := 1

	for cursor != "" {
		s.log.Infof("Fetching additional conversations batch %d", batchNumber)

		batch, nextCursor, err := getConversationsBatch(client, cursor)
		if err != nil {
			return nil, fmt.Errorf("slack client failed to get conversations batch %d: %w",
				batchNumber,
				err)
		}

		batchNumber++

		apiResults = append(apiResults, batch...)
		cursor = nextCursor
		s.log.Infof("Found %d conversations so far. Next cursor: %q", len(apiResults), cursor)
	}

	s.log.Infof("Found %d conversations total", len(apiResults))

	results := make([]Conversation, len(apiResults))
	for i, channel := range apiResults {
		results[i] = Conversation{
			Name: channel.Name,
			ID:   channel.ID,
		}
	}

	return results, nil
}

func getConversationsBatch(client SlackAPI, cursor string) ([]slack.Channel, string, error) {
	// TODO: Consider wiring a context into getConversationsBatch.
	ops := &slack.GetConversationsParameters{
		ExcludeArchived: "true",
		Cursor:          cursor,
	}

	apiResults, nextCursor, err := client.GetConversations(ops)
	if err != nil {
		return nil, "", fmt.Errorf("slack getConversationsBatch err: %w", err)
	}

	return apiResults, nextCursor, nil
}
