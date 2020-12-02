package reactjiupdate

import (
	"fmt"

	"github.com/cpu/gorfbot/botcmd"
	"github.com/cpu/gorfbot/config"
	"github.com/cpu/gorfbot/slack"
	"github.com/cpu/gorfbot/storage/models"
	"github.com/sirupsen/logrus"
)

const (
	handlerName = "reactji usage"
)

type reactjiHandler struct {
	log *logrus.Logger
}

func init() {
	botcmd.MustAddReactionHandler(&botcmd.ReactionCommand{
		Name:    handlerName,
		Handler: &reactjiHandler{},
	})
}

var errEmptyReaction = fmt.Errorf("%s handler error: empty emoji update received", handlerName)

func (rh reactjiHandler) Run(reaction *slack.Reaction, runCtx botcmd.RunContext) error {
	if reaction.Reaction == "" {
		return errEmptyReaction
	}

	initialCount := 1
	if reaction.Removed {
		initialCount = 0
	}

	e := models.Emoji{
		User:     reaction.User,
		Emoji:    reaction.Reaction,
		Count:    initialCount,
		Reaction: true,
	}

	updatedE, err := runCtx.Storage.UpsertEmojiCount(e, reaction.Removed)
	if err != nil {
		return fmt.Errorf("%s storage returned err: %w", handlerName, err)
	}

	user := runCtx.Slack.UserName(updatedE.User)

	if reaction.Removed {
		rh.log.Infof("%s update - User %q (%s) removed reactji %q (new history: %d times)",
			handlerName, user, updatedE.User, updatedE.Emoji, updatedE.Count-1)
	} else {
		rh.log.Infof("%s update - User %q (%s) reacted with reactji %q (history: %d times)",
			handlerName, user, updatedE.User, updatedE.Emoji, updatedE.Count+1)
	}

	return nil
}

func (rh *reactjiHandler) Configure(log *logrus.Logger, c *config.Config) error {
	rh.log = log
	return nil
}
