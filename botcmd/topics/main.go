package topics

import (
	"bytes"
	"flag"
	"fmt"

	"github.com/cpu/gorfbot/botcmd"
	"github.com/cpu/gorfbot/config"
	"github.com/cpu/gorfbot/storage"
	"github.com/sirupsen/logrus"
)

const (
	cmdName = "topics"
)

type topicsCmd struct {
	log *logrus.Logger
}

func init() {
	botcmd.MustAddCommand(&botcmd.BasicCommand{
		Name:        cmdName,
		Icon:        ":newspaper:",
		Description: "List previous channel topics",
		Handler:     &topicsCmd{},
	})
}

func (cmd topicsCmd) Run(text string, runCtx botcmd.RunContext) (botcmd.RunResult, error) {
	flagSet := flag.NewFlagSet(cmdName, flag.ContinueOnError)
	limit := flagSet.Int64("limit", 5, "optional limit for number of topics to display")
	channelFlag := flagSet.String("channel", "", "optional channel name to display topics for")
	asc := flagSet.Bool("asc", false, "list topics in ascending age")

	if respText := botcmd.ParseFlags(text, flagSet); respText != "" {
		return botcmd.RunResult{Message: respText}, nil
	}

	var channelID string

	var channelName string

	if *channelFlag != "" {
		channelID = runCtx.Slack.ConversationID(*channelFlag)
		if channelID == "" {
			return botcmd.RunResult{Message: "no such channel"}, nil
		}

		channelName = *channelFlag
	} else {
		channelID = runCtx.Message.ChannelID
		channelName = runCtx.Slack.ConversationName(channelID)
	}

	opts := storage.GetTopicOptions{
		Channel: channelID,
		FindOptions: storage.FindOptions{
			Limit:     *limit,
			Asc:       *asc,
			SortField: "date",
		},
	}
	cmd.log.Infof("Getting topics for opts %#v", opts)

	topics, err := runCtx.Storage.GetTopics(opts)
	if err != nil {
		return botcmd.RunResult{},
			fmt.Errorf("%s: failed to get topics from storage opts: %v err: %w",
				cmdName, opts, err)
	}

	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, ":newspaper: :mega: %d topics from channel *#%s* :mega: :newspaper:\n", len(topics), channelName)

	for _, topic := range topics {
		userName := runCtx.Slack.UserName(topic.Creator)
		// ignore date errs - if there's an err just use the empty time.Time
		date, _ := runCtx.Slack.ParseTimestamp(topic.Date)
		dateStr := botcmd.FormatTime(date)

		fmt.Fprintf(buf, "\t:rolled_up_newspaper: %s - Topic changed by _%s_ to :scroll: *%q*\n",
			dateStr, userName, topic.Topic)
	}

	return botcmd.RunResult{Message: buf.String()}, nil
}

func (cmd *topicsCmd) Configure(log *logrus.Logger, c *config.Config) error {
	cmd.log = log
	return nil
}
