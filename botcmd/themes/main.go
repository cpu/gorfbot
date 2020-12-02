package themes

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/cpu/gorfbot/botcmd"
	"github.com/cpu/gorfbot/config"
	"github.com/cpu/gorfbot/storage"
	"github.com/cpu/gorfbot/storage/models"
	"github.com/sirupsen/logrus"
)

const (
	cmdName = "themes"
)

var (
	addThemeRegex                = regexp.MustCompile(`([^#]+) ((?:#[\da-fA-F]{6}[\s\,]*){8})`)
	addThemeRegexExpectedMatches = 3
)

type themesCmd struct {
	log *logrus.Logger
}

func init() {
	botcmd.MustAddCommand(&botcmd.BasicCommand{
		Name:        cmdName,
		Icon:        ":art:",
		Description: "List saved Slack themes, add new ones",
		Handler:     &themesCmd{},
	})
}

func (cmd themesCmd) list(runCtx botcmd.RunContext) (botcmd.RunResult, error) {
	opts := storage.GetThemeOptions{
		FindOptions: storage.FindOptions{
			SortField: "name",
		},
	}

	themes, err := runCtx.Storage.GetThemes(opts)
	if err != nil {
		return botcmd.RunResult{}, fmt.Errorf("%s theme storage err: %w", cmdName, err)
	}

	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, ":art: %d saved themes :art:\n", len(themes))

	for i, t := range themes {
		creator := runCtx.Slack.UserName(t.Creator)
		fmt.Fprintf(buf, "%d. - Theme _*%s*_ by *%s*:\n%s\n",
			i+1, t.Name, creator, t.Theme)
	}

	return botcmd.RunResult{Message: buf.String()}, nil
}

func (cmd themesCmd) add(text string, runCtx botcmd.RunContext) (botcmd.RunResult, error) {
	matches := addThemeRegex.FindStringSubmatch(text)
	if len(matches) != addThemeRegexExpectedMatches {
		return botcmd.RunResult{
			Message: fmt.Sprintf(":interrobang: Usage `!%s add <themeName> <theme>`",
				cmdName),
		}, nil
	}

	if runCtx.Message == nil {
		return botcmd.RunResult{}, botcmd.ErrNilMessage
	}

	theme := models.Theme{
		Name:    matches[1],
		Theme:   matches[2],
		Creator: runCtx.Message.UserID,
	}
	creatorName := runCtx.Slack.UserName(theme.Creator)
	cmd.log.Infof("%s adding theme %q with name %q and creator %q (%s)\n",
		cmdName, theme.Theme, theme.Name, creatorName, theme.Creator)

	if err := runCtx.Storage.AddTheme(theme); err != nil {
		return botcmd.RunResult{}, fmt.Errorf("%s theme storage err: %w", cmdName, err)
	}

	return botcmd.RunResult{Reactji: []string{"art", "lower_left_paintbrush"}}, nil
}

func (cmd themesCmd) Run(text string, runCtx botcmd.RunContext) (botcmd.RunResult, error) {
	words := strings.Split(text, " ")
	subcmd := words[0]
	leftovers := strings.Join(words[1:], " ")

	if subcmd == "" || subcmd == "list" {
		return cmd.list(runCtx)
	} else if subcmd == "add" {
		return cmd.add(leftovers, runCtx)
	} else if subcmd == "-h" || subcmd == "--help" || subcmd == "help" {
		return botcmd.RunResult{
			Message: fmt.Sprintf(
				":speech_balloon: :bookmark_tabs: Usage of !*%s*: `!%s [list|add]`.\n"+
					"See `!%s list -h` and `!%s add -h` for more information.",
				cmdName, cmdName, cmdName, cmdName)}, nil
	}

	return botcmd.RunResult{
		Message: fmt.Sprintf(":interrobang: %q isn't a known %s subcommand? Try `!%s [add|list]`",
			subcmd, cmdName, cmdName),
	}, nil
}

func (cmd *themesCmd) Configure(log *logrus.Logger, c *config.Config) error {
	cmd.log = log
	return nil
}
