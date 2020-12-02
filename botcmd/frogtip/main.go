package frogtip

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/cpu/gorfbot/botcmd"
	"github.com/cpu/gorfbot/config"
	"github.com/sirupsen/logrus"
)

const (
	cmdName = "frogtip"
	apiURL  = "https://frog.tips/api/1/tips/"
)

var (
	defaultUserAgent = fmt.Sprintf("%s/0.0.1", cmdName)
	defaultTimeout   = time.Second * 30
)

type frogtipCmd struct {
	log *logrus.Logger
	api frogAPI
}

func init() {
	botcmd.MustAddCommand(&botcmd.BasicCommand{
		Name:        cmdName,
		Icon:        ":frog:",
		Description: "Frog care and feeding",
		Handler: &frogtipCmd{
			api: frogAPI{
				client: &http.Client{
					Timeout: defaultTimeout,
				},
			},
		},
	})
}

type frogAPI struct {
	client    frogClient
	userAgent string
}

type frogClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type errBadResponseCode struct {
	code int
}

func (e errBadResponseCode) Error() string {
	return fmt.Sprintf("%s cmd got non-200 response from tips API: %d",
		cmdName, e.code)
}

func (f frogAPI) GetTips() (*tipsResult, error) {
	ctx, cancel := config.ContextForTimeout(defaultTimeout)
	defer cancel()

	getTipReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)

	userAgent := defaultUserAgent
	if f.userAgent != "" {
		userAgent = f.userAgent
	}

	getTipReq.Header.Set("User-Agent", userAgent)

	resp, err := f.client.Do(getTipReq)
	if err != nil {
		return nil, fmt.Errorf("%s cmd error making get tips request: %w",
			cmdName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errBadResponseCode{resp.StatusCode}
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s cmd got err reading tips response: %w",
			cmdName, err)
	}

	var results tipsResult
	if err := json.Unmarshal(bodyBytes, &results); err != nil {
		return nil, fmt.Errorf("%s cmd hit err unmarshaling tips response: %w",
			cmdName, err)
	}

	return &results, nil
}

type tipsResult struct {
	Tips []tip
}

type tip struct {
	Tip    string
	Number int
}

var errNoTips = fmt.Errorf("%s cmd tip API had no tips :-(", cmdName)

func (cmd frogtipCmd) Run(text string, runCtx botcmd.RunContext) (botcmd.RunResult, error) {
	start := time.Now()

	tipResults, err := cmd.api.GetTips()
	if err != nil {
		return botcmd.RunResult{}, err
	}

	elapsed := time.Since(start)
	cmd.log.Infof("%s fetched %d tips in %s", cmdName, len(tipResults.Tips), elapsed)

	if len(tipResults.Tips) == 0 {
		return botcmd.RunResult{}, errNoTips
	}

	tip := strings.ReplaceAll(tipResults.Tips[0].Tip, `\"`, `"`)
	tipMessage := fmt.Sprintf(":frog: :speech_balloon: %q", tip)

	return botcmd.RunResult{
		Message: tipMessage,
		Reactji: []string{"yin_yang", "pray"},
	}, nil
}

func (cmd *frogtipCmd) Configure(log *logrus.Logger, c *config.Config) error {
	cmd.log = log
	if c != nil {
		cmd.api.userAgent = c.FrogtipConf.UserAgent
	}

	return nil
}
