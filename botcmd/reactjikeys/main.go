package reactjikeys

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/cpu/gorfbot/botcmd"
	"github.com/cpu/gorfbot/config"
	"github.com/sirupsen/logrus"
)

const (
	patternName                  = "reactji keywords"
	patternRegexp                = `(\w+)`
	patternExpectedSubmatchCount = 2
)

type reactjikeysPattern struct {
	log    *logrus.Logger
	config config.ReactjiKeysConfig
}

func init() {
	botcmd.MustAddPattern(&botcmd.PatternCommand{
		Name:    patternName,
		Handler: &reactjikeysPattern{},
		Pattern: regexp.MustCompile(patternRegexp),
	})
}

var (
	errAllSubmatchesEmpty = fmt.Errorf("%s pattern executed with no submatches", patternName)
)

type errUnexpectedSubmatchLen struct {
	got interface{}
}

func (e errUnexpectedSubmatchLen) Error() string {
	return fmt.Sprintf("%s pattern found submatch with unexpected len: %#v",
		patternName, e.got)
}

func (p reactjikeysPattern) Run(allSubmatches [][]string, runCtx botcmd.RunContext) (botcmd.RunResult, error) {
	if len(allSubmatches) < 1 {
		return botcmd.RunResult{}, errAllSubmatchesEmpty
	}

	reactionsMap := make(map[string]bool)

	for _, submatches := range allSubmatches {
		if len(submatches) != patternExpectedSubmatchCount {
			return botcmd.RunResult{}, errUnexpectedSubmatchLen{submatches}
		}

		word := strings.ToLower(submatches[1])

		if reactions, found := p.config.Keywords[word]; found {
			p.log.Infof("word: %q triggers %#v", word, reactions)

			for _, r := range reactions {
				reactionsMap[r] = true
			}
		}
	}

	var allReactions []string //nolint:prealloc
	for reaction := range reactionsMap {
		allReactions = append(allReactions, reaction)
	}

	sort.Strings(allReactions) // Sort reactions to make unit tests easier.

	return botcmd.RunResult{
		Reactji: allReactions,
	}, nil
}

func (p *reactjikeysPattern) Configure(log *logrus.Logger, c *config.Config) error {
	p.log = log
	if c != nil {
		p.config = c.ReactjiKeysConf
		p.log.Tracef("Loaded reactji config: %v", p.config)
	}

	return nil
}
