//nolint:misspell
package mktheme

import (
	"flag"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/cpu/gorfbot/botcmd"
	"github.com/cpu/gorfbot/config"
	petname "github.com/dustinkirkland/golang-petname"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/sirupsen/logrus"
)

const (
	cmdName         = "mktheme"
	themeColours    = 8
	themePetnameLen = 2
)

type mkthemeCmd struct {
	log *logrus.Logger
}

type slackTheme struct {
	name        string
	paletteType string
	colours     [themeColours]colorful.Color
}

type errUnknownPalette struct {
	palette string
}

func (e errUnknownPalette) Error() string {
	return fmt.Sprintf("unknown palette type %q", e.palette)
}

//nolint:gosec, gomnd
func (t *slackTheme) Generate() error {
	t.name = petname.Generate(themePetnameLen, " ")

	var err error

	var colours []colorful.Color

	switch t.paletteType {
	case "none":
		for i := 0; i < themeColours; i++ {
			colours = append(colours, colorful.Hcl(
				rand.Float64()*360.0, rand.Float64(), rand.Float64()*0.4))
		}
	case "warm":
		colours, err = colorful.WarmPalette(themeColours)
	case "happy":
		colours, err = colorful.HappyPalette(themeColours)
	case "soft":
		colours, err = colorful.SoftPalette(themeColours)
	default:
		err = errUnknownPalette{t.paletteType}
	}

	if err != nil {
		return err
	}

	copy(t.colours[:], colours[:themeColours])

	return nil
}

func (t slackTheme) String() string {
	encoded := make([]string, len(t.colours))
	for i, c := range t.colours {
		encoded[i] = c.Hex()
	}

	var palette string

	if t.paletteType != "" {
		palette = t.paletteType + " "
	}

	return fmt.Sprintf(":lower_left_paintbrush: *%s%s*:\n%s",
		palette, t.name, strings.Join(encoded, ","))
}

func init() {
	botcmd.MustAddCommand(&botcmd.BasicCommand{
		Name:        cmdName,
		Icon:        ":lower_left_paintbrush:",
		Description: "Generate a new Slack theme",
		Handler:     &mkthemeCmd{},
	})
}

func (cmd mkthemeCmd) Run(text string, runCtx botcmd.RunContext) (botcmd.RunResult, error) {
	flagSet := flag.NewFlagSet(cmdName, flag.ContinueOnError)
	paletteFlag := flagSet.String("palette", "warm", `Palette type: [none, warm, happy, soft]`)

	if respText := botcmd.ParseFlags(text, flagSet); respText != "" {
		return botcmd.RunResult{Message: respText}, nil
	}

	theme := &slackTheme{
		paletteType: strings.ToLower(*paletteFlag),
	}

	cmd.log.Infof("%s making a theme with palette type %q", cmdName, theme.paletteType)

	if err := theme.Generate(); err != nil {
		return botcmd.RunResult{Message: err.Error()}, nil
	}

	return botcmd.RunResult{Message: theme.String()}, nil
}

func (cmd *mkthemeCmd) Configure(log *logrus.Logger, c *config.Config) error {
	cmd.log = log

	if c != nil && c.MkthemeConf.RandomSeed > 0 {
		rand.Seed(c.MkthemeConf.RandomSeed)
	} else {
		rand.Seed(time.Now().UnixNano())
	}

	return nil
}
