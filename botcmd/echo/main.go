package echo

import (
	"fmt"

	"github.com/cpu/gorfbot/botcmd"
	"github.com/cpu/gorfbot/config"
	"github.com/sirupsen/logrus"
)

const (
	cmdName = "echo"
)

type echoCmd struct {
	log *logrus.Logger
}

func init() {
	botcmd.MustAddCommand(&botcmd.BasicCommand{
		Name:        cmdName,
		Icon:        ":repeat:",
		Description: "Have Garfbot echo a message",
		Handler:     &echoCmd{},
	})
}

func (cmd echoCmd) Run(text string, runCtx botcmd.RunContext) (botcmd.RunResult, error) {
	if runCtx.Message == nil {
		return botcmd.RunResult{},
			fmt.Errorf("%s cmd error: %w", cmdName, botcmd.ErrNilMessage)
	}

	return botcmd.RunResult{Message: text}, nil
}

func (cmd *echoCmd) Configure(log *logrus.Logger, c *config.Config) error {
	cmd.log = log
	return nil
}
