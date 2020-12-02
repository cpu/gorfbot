package panoptimoji

import (
	"fmt"
	"regexp"

	"github.com/cpu/gorfbot/botcmd"
	"github.com/cpu/gorfbot/config"
	"github.com/cpu/gorfbot/storage/models"
	"github.com/sirupsen/logrus"
)

const (
	patternName = "emoji usage"
	// Custom emoji are ':' and ':' surrounding a custom emoji name.
	// The name component must be:
	//   * Lowercase a-z, with limited punctuation (-,_,+,')
	//   * No more than 100 characters in length
	patternRegexp              = `(\:[a-z0-9\-\_\+\']{1,100}\:)`
	expectedAllSubmatchesCount = 1
	expectedSubmatchesCount    = 2
)

type panoptimojiPattern struct {
	log *logrus.Logger
}

func init() {
	botcmd.MustAddPattern(&botcmd.PatternCommand{
		Name:    patternName,
		Handler: &panoptimojiPattern{},
		Pattern: regexp.MustCompile(patternRegexp),
	})
}

var (
	errNoSubmatches = fmt.Errorf("%s pattern executed with no submatches", patternName)
)

type errTooFewSubmatches struct {
	actual []string
}

func (e errTooFewSubmatches) Error() string {
	return fmt.Sprintf("%s expected two submatches found %v", patternName, e.actual)
}

func (p panoptimojiPattern) Run(allSubmatches [][]string, runCtx botcmd.RunContext) (botcmd.RunResult, error) {
	if runCtx.Message == nil {
		return botcmd.RunResult{},
			fmt.Errorf("%s pattern error: %w", patternName, botcmd.ErrNilMessage)
	}

	if len(allSubmatches) < expectedAllSubmatchesCount {
		return botcmd.RunResult{}, errNoSubmatches
	}

	for _, submatch := range allSubmatches {
		if len(submatch) != expectedSubmatchesCount {
			return botcmd.RunResult{}, errTooFewSubmatches{submatch}
		}

		e := models.Emoji{
			User:  runCtx.Message.UserID,
			Emoji: submatch[1],
			Count: 1,
		}

		updatedE, err := runCtx.Storage.UpsertEmojiCount(e, false)
		if err != nil {
			return botcmd.RunResult{},
				fmt.Errorf("%s storage returned err: %w", patternName, err)
		}

		user := runCtx.Slack.UserName(updatedE.User)
		p.log.Infof("%s update - User %q (%s) has used emoji %q (history: %d times)",
			patternName, user, updatedE.User, updatedE.Emoji, updatedE.Count+1)
	}

	return botcmd.RunResult{}, nil
}

func (p *panoptimojiPattern) Configure(log *logrus.Logger, c *config.Config) error {
	p.log = log
	return nil
}
