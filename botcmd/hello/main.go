package hello

import (
	"fmt"

	"github.com/cpu/gorfbot/botcmd"
	"github.com/cpu/gorfbot/config"
	"github.com/sirupsen/logrus"
)

const (
	cmdName = "hello"
)

type helloCmd struct {
	log *logrus.Logger
}

func init() {
	botcmd.MustAddCommand(&botcmd.BasicCommand{
		Name:        cmdName,
		Icon:        ":wave:",
		Description: "Say hello to Gorf",
		Handler:     &helloCmd{},
	})
}

func (cmd helloCmd) Run(text string, runCtx botcmd.RunContext) (botcmd.RunResult, error) {
	if runCtx.Message == nil {
		return botcmd.RunResult{},
			fmt.Errorf("%s cmd error: %w", cmdName, botcmd.ErrNilMessage)
	}

	return botcmd.RunResult{
		Message: "hello!",
		Reactji: []string{"wave"},
	}, nil
}

func (cmd *helloCmd) Configure(log *logrus.Logger, c *config.Config) error {
	cmd.log = log
	return nil
}
