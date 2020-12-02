package bot

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/cpu/gorfbot/botcmd"

	// Import commands so that each package's init() is run.
	_ "github.com/cpu/gorfbot/botcmd/echo"
	_ "github.com/cpu/gorfbot/botcmd/emoji"
	_ "github.com/cpu/gorfbot/botcmd/frogtip"
	_ "github.com/cpu/gorfbot/botcmd/gis"
	_ "github.com/cpu/gorfbot/botcmd/hello"
	_ "github.com/cpu/gorfbot/botcmd/mktheme"
	_ "github.com/cpu/gorfbot/botcmd/panoptimoji"
	_ "github.com/cpu/gorfbot/botcmd/rarepattern"
	_ "github.com/cpu/gorfbot/botcmd/reactjikeys"
	_ "github.com/cpu/gorfbot/botcmd/reactjiupdate"
	_ "github.com/cpu/gorfbot/botcmd/themes"
	_ "github.com/cpu/gorfbot/botcmd/topics"
	_ "github.com/cpu/gorfbot/botcmd/topicupdate"
	"github.com/cpu/gorfbot/config"
	"github.com/cpu/gorfbot/slack"
	"github.com/cpu/gorfbot/storage"
	"github.com/cpu/gorfbot/storage/mongo"
	"github.com/sirupsen/logrus"
)

// Bot's know how to run and not much else.
type Bot interface {
	// Start the bot. Does not return. Call from a goroutine or block forever.
	Run()
}

type botImpl struct {
	log      *logrus.Logger
	storage  storage.Storage
	slack    slack.Client
	registry *botcmd.CommandRegistry
}

func New(log *logrus.Logger, c *config.Config) (Bot, error) {
	if log == nil {
		log = logrus.New()
	}

	if c == nil {
		return nil, fmt.Errorf("bot error: %w", config.ErrNilConfig)
	}

	// Let's build a Gorfbot
	bot := &botImpl{
		log:      log,
		registry: botcmd.DefaultRegistry,
	}

	// Connect to mongo storage
	storage, err := mongo.NewMongoStorage(log, c)
	if err != nil {
		return nil, fmt.Errorf("bot storage error: %w", err)
	}

	bot.storage = storage

	// Connect to slack
	slack, err := slack.New(log, c)
	if err != nil {
		return nil, fmt.Errorf("bot slack error: %w", err)
	}

	bot.slack = slack

	// Configure each command/pattern/reaction handler
	for _, cmd := range bot.registry.GetConfigurables() {
		if cmd == nil {
			panic("nil configurable registered???\n")
		}

		if err := cmd.Configure(log, c); err != nil {
			return nil, fmt.Errorf("bot cmd configure error: %w", err)
		}
	}

	// Ready to Run()
	return bot, nil
}

// Run forever.
func (b botImpl) Run() {
	// Start consuming messages and reactions
	msgChan := make(chan *slack.Message)
	reactionChan := make(chan *slack.Reaction)

	go b.slack.Listen(msgChan, reactionChan)

	// Process messages forever
	for {
		select {
		case msg := <-msgChan:
			if msg == nil {
				continue
			}

			b.log.Tracef("msg: %q", msg.Text)
			// First try the message through all of the configured pattern commands.
			b.tryMessageAsPattern(msg)
			// Then try to treat the message as a bot command.
			b.tryMessageAsCommand(msg)
		case reaction := <-reactionChan:
			if reaction == nil {
				continue
			}
			// Feed the reaction through the reaction handlers
			b.tryReactionHandlers(reaction)
		}
	}
}

func (b botImpl) runCtx(m *slack.Message) botcmd.RunContext {
	return botcmd.RunContext{
		Message: m,
		Storage: b.storage,
		Slack:   b.slack,
	}
}

// tryReactionHandlers calls Run on each reaction handler with the provided
// reaction.
func (b botImpl) tryReactionHandlers(reaction *slack.Reaction) {
	for _, handler := range b.registry.GetReactionHandlers() {
		if err := handler.Handler.Run(reaction, b.runCtx(nil)); err != nil {
			b.log.Errorf("Reaction handler %q returned an error: %v", handler.Name, err)
		}
	}
}

// handleRunResult processes the optional message and reactions returned by
// a botcmd.
func (b botImpl) handleRunResult(m *slack.Message, res botcmd.RunResult) {
	if res.Message != "" {
		b.log.Tracef("Posting returned msg %q", res.Message)
		b.slack.SendMessage(res.Message, m.ChannelID)
	}

	b.addReactions(res.Reactji, m)
}

// addReactions adds a list of reactions to a given message.
func (b botImpl) addReactions(reactions []string, m *slack.Message) {
	if len(reactions) > 0 {
		b.log.Infof("Adding reactions: %q", reactions)
	}

	for _, reactji := range reactions {
		if err := b.slack.AddReaction(reactji, m); err != nil {
			b.log.Errorf("Failed to add reaction: %v", err)
		}
	}
}

// tryMessageAsPattern tries to match a message on any of the configured pattern
// handlers, calling Run() on handlers that have a pattern match.
func (b botImpl) tryMessageAsPattern(m *slack.Message) {
	// Try every pattern's regex and call Run() for any that match.
	for _, pattern := range b.registry.GetPatterns() {
		if matches := pattern.Pattern.FindAllStringSubmatch(m.Text, -1); len(matches) > 0 {
			b.log.Infof("pattern %q matched with %q", pattern.Name, pattern.Pattern)

			if res, err := pattern.Handler.Run(matches, b.runCtx(m)); err != nil {
				b.log.Errorf("Pattern %q returned an error: %v", pattern.Name, err)

				continue // pattern returned an error
			} else {
				b.handleRunResult(m, res)
			}
		}
	}
}

// tryMessageAsCommand tries to process a received message as if it were a bot cmd,
// being flexible about how users might try to use commands.
func (b botImpl) tryMessageAsCommand(m *slack.Message) {
	// Split the incoming message text. It should have at least two words in it
	// for it to be a command to handle.
	textWords := strings.Split(m.Text, " ")

	firstWord := textWords[0]
	// Does the first word start with '!'?
	hasCmdPrefix := strings.HasPrefix(firstWord, "!")
	// Does the first word start with '<@'?
	hasMentionPrefix := strings.HasPrefix(firstWord, "<@")
	// If it isn't a cmd or a mention that could be a command then return.
	if !hasCmdPrefix && !hasMentionPrefix {
		b.log.Trace("Received message didn't have cmd prefix or start with a mention")
		return
	}

	b.log.Infof("Processing potential command message, first word: %q hasCmdPrefix: %v hasMentionPrefix: %v\n",
		firstWord, hasCmdPrefix, hasMentionPrefix)

	if hasCmdPrefix {
		// Process as a bare cmd heard in a channel.
		cmd := strings.TrimPrefix(firstWord, "!")
		rest := strings.Join(textWords[1:], " ")
		b.log.Infof("Processing heard cmd: %q with rest %q\n", cmd, rest)
		b.handleCommandMessage(cmd, rest, m)
	} else if hasMentionPrefix {
		// Process as a @ mention heard in a channel.
		// The mention must be to the bot.
		mention := firstWord
		expected := fmt.Sprintf("<@%s>", b.slack.BotID())
		if mention != expected {
			b.log.Infof("Message mention wasn't to bot: Got %q expected %q",
				mention, expected)

			return
		}
		// There must be a command word after the mention for it to be worth
		// processing.
		if len(textWords) < 2 { // nolint:gomnd
			b.log.Info("Message mention too short to be a command message")

			return
		}
		cmd := strings.TrimPrefix(textWords[1], "!")
		rest := strings.Join(textWords[2:], " ")
		b.log.Infof("Processing mentioned cmd: %q with rest %q\n", cmd, rest)
		b.handleCommandMessage(cmd, rest, m)
	}
}

// handleCommandMessage tries to find a registered command with the given cmdName
// and runs it with the rest of the message.
func (b botImpl) handleCommandMessage(cmdName string, rest string, m *slack.Message) {
	if cmdName == "" {
		b.log.Warn("Got empty command name in handleCommandMessage")

		return
	}

	if cmdName == "help" || cmdName == "-h" || cmdName == "--help" {
		b.botHelp(m)

		return
	}

	if cmd := b.registry.GetCommand(cmdName); cmd == nil {
		b.log.Warnf("Command %q not registered with bot", cmdName)
		b.addReactions([]string{"interrobang"}, m)

		return // command not known
	} else if res, err := cmd.Handler.Run(rest, b.runCtx(m)); err != nil {
		b.log.Errorf("Command %q returned an error: %v", cmdName, err)
		b.addReactions([]string{"negative_squared_cross_mark"}, m)

		return // command returned an error
	} else {
		b.handleRunResult(m, res)
	}
}

// botHelp enumerates the configured botcmds and pattern/reaction handlers
// and posts help information in reply to the given message.
//nolint:lll
func (b botImpl) botHelp(m *slack.Message) {
	// TODO: template this mess.
	userName := b.slack.UserName(m.UserID)
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, ":wave: Hello %s\n", userName)
	fmt.Fprintf(buf, ":speech_balloon: I'm *Gorfbot* - Here are the commands I know:\n")

	for _, cmd := range b.registry.GetCommands() {
		fmt.Fprintf(buf, "\t\t :three_button_mouse: %s `!%s` - %s\n", cmd.Icon, cmd.Name, cmd.Description)
	}

	fmt.Fprintf(buf, ":speech_balloon: I'm also keeping track of\n")

	for _, pattern := range b.registry.GetPatterns() {
		fmt.Fprintf(buf, "\t\t :eyes: _%s_\n", pattern.Name)
	}

	for _, handler := range b.registry.GetReactionHandlers() {
		fmt.Fprintf(buf, "\t\t :eyes: _%s_\n", handler.Name)
	}

	fmt.Fprintf(buf, ":speech_balloon: - To run a command say `!<command> [arguments]` in a channel/conversation that we're both in.\n")
	fmt.Fprintf(buf, ":speech_balloon: - Most commands offer help, try `!<command> -h`, like `!emoji -h`\n")
	fmt.Fprintf(buf, ":nose: :kissing_cat: Smell ya later!")
	b.handleRunResult(m, botcmd.RunResult{Message: buf.String()})
}
