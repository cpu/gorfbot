package emoji

import (
	"bytes"
	"flag"
	"fmt"
	"strings"

	"github.com/cpu/gorfbot/botcmd"
	"github.com/cpu/gorfbot/config"
	"github.com/cpu/gorfbot/storage"
	"github.com/sirupsen/logrus"
)

const (
	cmdName = "emoji"
)

type emojiCmd struct {
	log *logrus.Logger
}

func init() {
	botcmd.MustAddCommand(&botcmd.BasicCommand{
		Name:        cmdName,
		Icon:        ":upside_down_face:",
		Description: "Find someone's most used emoji/reactji",
		Handler:     &emojiCmd{},
	})
}

// TODO: Template this gnarly output building.
//nolint:nestif,funlen
func (cmd emojiCmd) Run(text string, runCtx botcmd.RunContext) (botcmd.RunResult, error) {
	flagSet := flag.NewFlagSet(cmdName, flag.ContinueOnError)
	limit := flagSet.Int64("limit", 5, "limit for number of emoji to display")
	asc := flagSet.Bool("asc", false, "list topics in order of ascending usage count")
	emojiFlag := flagSet.String("emoji", "", "display count only for matching emoji")
	usernameFlag := flagSet.String("user", "", "display emoji stats for a user other than yourself")
	reactions := flagSet.Bool("reactions", false, "only include reactions stats")

	if respText := botcmd.ParseFlags(text, flagSet); respText != "" {
		return botcmd.RunResult{Message: respText}, nil
	}

	var userID string

	var username string

	if *usernameFlag == "" {
		userID = runCtx.Message.UserID
		username = runCtx.Slack.UserName(userID)
	} else {
		username = *usernameFlag
		userID = runCtx.Slack.UserID(username)
	}

	opts := storage.GetEmojiOptions{
		FindOptions: storage.FindOptions{
			SortField: "count",
			Limit:     *limit,
			Asc:       *asc,
		},
		User:     userID,
		Emoji:    *emojiFlag,
		Reaction: *reactions,
	}
	cmd.log.Infof("Getting emoji with options: %#v", opts)

	emoji, err := runCtx.Storage.GetEmoji(opts)
	if err != nil {
		return botcmd.RunResult{},
			fmt.Errorf("%s: failed to get emoji from storage opts: %v err: %w",
				cmdName, opts, err)
	}

	buf := new(bytes.Buffer)

	if *emojiFlag != "" {
		if len(emoji) == 0 {
			fmt.Fprintf(buf, "%s has not been observed using emoji %q\n",
				username, *emojiFlag)
		} else {
			emojiMatch := emoji[0]
			fmt.Fprintf(buf, "%s has used the %s emoji %d times\n",
				username, *emojiFlag, emojiMatch.Count)
		}
	} else {
		header := "Top"
		if *asc {
			header = "Rarest"
		}
		objects := "emoji"
		if *reactions {
			objects = "reactji"
		}
		fmt.Fprintf(buf, ":upside_down_face: %s %d observed %s for *%s*:\n", header, len(emoji), objects, username)
		for _, e := range emoji {
			if !strings.HasPrefix(e.Emoji, ":") {
				e.Emoji = ":" + e.Emoji
			}
			if !strings.HasSuffix(e.Emoji, ":") {
				e.Emoji += ":"
			}
			fmt.Fprintf(buf, "\t%s - used _%d times_.\n", e.Emoji, e.Count)
		}
	}

	return botcmd.RunResult{Message: buf.String()}, nil
}

func (cmd *emojiCmd) Configure(log *logrus.Logger, c *config.Config) error {
	cmd.log = log
	return nil
}
