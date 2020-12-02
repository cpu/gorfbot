package topicupdate

import (
	"fmt"
	"regexp"

	"github.com/cpu/gorfbot/botcmd"
	"github.com/cpu/gorfbot/config"
	"github.com/cpu/gorfbot/storage/models"
	"github.com/sirupsen/logrus"
)

const (
	patternName                      = "topic updates"
	topicUpdateRegexp                = `^<@([\w]+)> set the channel topic: (.*)$`
	topicUpdateExpectedSubmatchCount = 3
)

type submatchErr struct {
	msg string
	got interface{}
}

func (e submatchErr) Error() string {
	return fmt.Sprintf("%s pattern error: %s, got %v", patternName, e.msg, e.got)
}

type topicUpdatePattern struct {
	log *logrus.Logger
}

func init() {
	botcmd.MustAddPattern(&botcmd.PatternCommand{
		Name:    patternName,
		Handler: &topicUpdatePattern{},
		Pattern: regexp.MustCompile(topicUpdateRegexp),
	})
}

func (p topicUpdatePattern) Run(allSubmatches [][]string, runCtx botcmd.RunContext) (botcmd.RunResult, error) {
	if runCtx.Message == nil {
		return botcmd.RunResult{},
			fmt.Errorf("%s pattern error: %w", patternName, botcmd.ErrNilMessage)
	}

	if len(allSubmatches) != 1 {
		return botcmd.RunResult{}, submatchErr{
			msg: "expected one submatch", got: allSubmatches}
	}

	submatches := allSubmatches[0]
	if len(submatches) < topicUpdateExpectedSubmatchCount {
		return botcmd.RunResult{}, submatchErr{
			msg: "too few submatches", got: submatches}
	}

	model := models.Topic{
		Channel: runCtx.Message.ChannelID,
		Date:    runCtx.Message.Timestamp,
		Creator: submatches[1],
		Topic:   submatches[2],
	}

	if err := runCtx.Storage.AddTopic(model); err != nil {
		return botcmd.RunResult{},
			fmt.Errorf("%s pattern error storing new topic: %w", patternName, err)
	}

	p.log.Info(model)

	return botcmd.RunResult{
		Reactji: []string{"mag", "newspaper"},
	}, nil
}

func (p *topicUpdatePattern) Configure(log *logrus.Logger, c *config.Config) error {
	p.log = log
	return nil
}
