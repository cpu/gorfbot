package botcmd

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"regexp"
	"strings"

	"github.com/cpu/gorfbot/config"
	"github.com/cpu/gorfbot/slack"
	"github.com/cpu/gorfbot/storage"
	"github.com/sirupsen/logrus"
)

var ErrNilMessage = errors.New("message was nil")

// RunContext binds together all of the contextual information a botcmd's Run function
// might need. Typically this is a message instance, a handle to storage, and
// a handle to a slack client to interact with.
type RunContext struct {
	Message *slack.Message
	Storage storage.Storage
	Slack   slack.Client
}

// RunResult is returned by a botcmd's Run function and can be used as a simple way
// to post a reply message and/or add reactions to the message that caused the
// botcmd to be run.
type RunResult struct {
	Message string
	Reactji []string
}

// Configurable is a common interface for anything (cmd, pattern cmd, reaction
// handler, etc) that can be configured with a logger and config instance.
type Configurable interface {
	// Configure is called with a log and config instance once they are available and
	// before any Run calls are made.
	Configure(log *logrus.Logger, c *config.Config) error
}

// PatternHandler describes a configurable that expects to be run with all
// submatches of a regex pattern.
//go:generate mockgen -destination=mocks/mock_pattern_handler.go -package=mocks . PatternHandler
type PatternHandler interface {
	Configurable
	// Run accepts a list of all of the submatches from a regexp and a runCtx.
	Run(allSubmatches [][]string, runCtx RunContext) (RunResult, error)
}

// PatternCommand describes a pattern handler and its associated pattern. Any messages
// that match the pattern will have the handler's Run function invoked with all
// of the submatches.
type PatternCommand struct {
	// Name of the PatternCommand. NOTE: used in help output.
	Name string
	// PatternHandler to invoke with matches.
	Handler PatternHandler
	// Pattern regexp. Messages matching this pattern will have the Handler invoked.
	Pattern *regexp.Regexp
}

// CommandHandler describes a configurable that expects to be run with some text
// when the associated basic command is invoked.
//go:generate mockgen -destination=mocks/mock_command_handler.go -package=mocks . CommandHandler
type CommandHandler interface {
	Configurable
	// Run is called for command matches with the text that appeared in the
	// message after the command name and a run context.
	Run(text string, runCtx RunContext) (RunResult, error)
}

// BasicCommand describes a basic bot command that can be invoked on demand with
// "!<cmd name>". Remaining text is passed to the command handler so that it can
// be parsed further (e.g. to have CLI options for the command).
type BasicCommand struct {
	// Name is the name of the command and how it is invoked.
	// TODO: Support aliases.
	Name string
	// Description is a short description of the command for help output.
	Description string
	// Icon is an emoji (no ":" delimiters) to use for this command in help
	// output.
	Icon string
	// Handler is a CommandHandler invoked when a command invocation for Name is
	// performed by a user.
	Handler CommandHandler
}

// ReactionHandler describes a configurable that has its Run function called when
// reactions are added/removed.
//go:generate mockgen -destination=mocks/mock_pattern_handler.go -package=mocks . PatternHandler
type ReactionHandler interface {
	Configurable
	// Run is called when a reaction is added or removed.
	Run(reaction *slack.Reaction, runCtx RunContext) error
}

// ReactionCommand describes a named reacton handler that is run when reactions
// are added/removed.
type ReactionCommand struct {
	// Name of the reaction handler. Used in help, should describe purpose of handler.
	Name string
	// Handler is invoked when reactions are added/removed.
	Handler ReactionHandler
}

// bufferedHelp returns a function that when invoked will write a help string
// for each of the flagset's flags to a returned byte buffer. This is an easy
// way to dynamically build up a help string for a flag set that can be sent to
// Slack as a message.
func bufferedHelp(flagSet *flag.FlagSet) (func(), *bytes.Buffer) {
	helpBuffer := new(bytes.Buffer)

	return func() {
		fmt.Fprintf(helpBuffer, ":speech_balloon: :bookmark_tabs: Usage of !*%s*:\n", flagSet.Name())
		flagSet.VisitAll(func(f *flag.Flag) {
			fmt.Fprintf(helpBuffer, "\t`-%s`\t%v (Default: `-%s %v`)\n",
				f.Name, f.Usage, f.Name, f.Value)
		})
	}, helpBuffer
}

// parseFlags tries to parse the given text with the given flagset. It sets the
// flagset's Usage function to a bufferedHelp instance and when help is requested
// returns the buffer.
func ParseFlags(text string, flagSet *flag.FlagSet) string {
	helpFunc, helpBuffer := bufferedHelp(flagSet)
	flagSet.Usage = helpFunc

	if err := flagSet.Parse(strings.Split(text, " ")); err != nil && !errors.Is(err, flag.ErrHelp) {
		return fmt.Sprintf("%s: failed to parse %q: %s", flagSet.Name(), text, err)
	} else if errors.Is(err, flag.ErrHelp) {
		return helpBuffer.String()
	}

	return ""
}
