package rarepattern

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/cpu/gorfbot/botcmd"
	"github.com/cpu/gorfbot/config"
	"github.com/cpu/gorfbot/storage/models"
	"github.com/sirupsen/logrus"
)

const (
	patternName         = "URLs"
	slackURLPattern     = `<([^|]+)(?:\|[^>]+)?>`
	expectedSubmatchLen = 2
)

type urlPattern struct {
	HostRegexp *regexp.Regexp
	PathRegexp *regexp.Regexp
	Collection string
	FirstMsg   string
	Reactji    []string
}

func (u urlPattern) Matches(log *logrus.Logger, url *url.URL) string {
	if u.PathRegexp == nil {
		return "" // Shouldn't happen
	}

	if !u.HostRegexp.MatchString(url.Host) {
		log.Tracef("URL Host %q doesn't match HostRegexp %q",
			url.Host, u.HostRegexp)
		return ""
	}

	log.Tracef("URL Host %q matches HostRegexp %q",
		url.Host, u.HostRegexp)

	if u.PathRegexp != nil {
		if !u.PathRegexp.MatchString(url.Path) {
			log.Tracef("URL Path %q doesn't match PathRegexp %q",
				url.Path, u.PathRegexp)
			return ""
		}

		log.Tracef("URL Path %q matches PathRegexp %v",
			url.Path, u.PathRegexp)
	}

	return u.Collection // Match
}

type rareURLPattern struct {
	log         *logrus.Logger
	urlPatterns []urlPattern
	config      config.URLsConfig
}

func init() {
	botcmd.MustAddPattern(&botcmd.PatternCommand{
		Name:    patternName,
		Handler: &rareURLPattern{},
		Pattern: regexp.MustCompile(slackURLPattern),
	})
}

type errBadMatches struct {
	msg string
	got interface{}
}

func (e errBadMatches) Error() string {
	return fmt.Sprintf("%s pattern error: %s, got %v",
		patternName, e.msg, e.got)
}

//nolint:funlen
func (p rareURLPattern) Run(allSubmatches [][]string, runCtx botcmd.RunContext) (botcmd.RunResult, error) {
	if len(allSubmatches) == 0 {
		return botcmd.RunResult{}, errBadMatches{"expected at least one submatch", allSubmatches}
	}

	var messages []string

	reactionsMap := make(map[string]bool)

	for _, submatches := range allSubmatches {
		if submatchLen := len(submatches); submatchLen != expectedSubmatchLen {
			return botcmd.RunResult{}, errBadMatches{
				fmt.Sprintf("unexpected submatch length: %d", submatchLen),
				submatches,
			}
		}

		urlPart := submatches[1]

		url, err := url.Parse(urlPart)
		if err != nil || url == nil {
			p.log.Warnf("%s pattern submatch part %q didn't parse as URL: %v",
				patternName, urlPart, err)
			return botcmd.RunResult{}, nil //
		}

		p.log.Infof("%s pattern saw URL for Host %q Path %q",
			patternName, url.Host, url.Path)

		for _, urlPattern := range p.urlPatterns {
			if collection := urlPattern.Matches(p.log, url); collection != "" {
				// Clear out the query and fragment before storing the count
				url.RawQuery = ""
				url.Fragment = ""
				u := models.URLCount{
					URL:         url.String(),
					Occurrences: 1,
				}

				updatedU, err := runCtx.Storage.UpsertURLCount(collection, u)
				if err != nil {
					return botcmd.RunResult{},
						fmt.Errorf("%s storage returned err: %w", patternName, err)
				}

				p.log.Infof("%s update - collection %q matched URL %q (history: %d times)",
					patternName, collection, updatedU.URL, updatedU.Occurrences+1)

				for _, r := range urlPattern.Reactji {
					reactionsMap[r] = true
				}

				if updatedU.Occurrences == 0 && urlPattern.FirstMsg != "" {
					messages = append(messages, urlPattern.FirstMsg)
				}
			}
		}
	}

	var reactions []string //nolint:prealloc
	for r := range reactionsMap {
		reactions = append(reactions, r)
	}

	sort.Strings(reactions)

	return botcmd.RunResult{
		Message: strings.Join(messages, "\n"),
		Reactji: reactions,
	}, nil
}

type errEmpty struct {
	what string
}

func (e errEmpty) Error() string {
	return fmt.Sprintf("error: URL in urlsconf with empty %s", e.what)
}

func newURLPattern(urlConf config.URLConfig) (urlPattern, error) {
	if urlConf.HostPattern == "" {
		return urlPattern{}, errEmpty{"hostpattern"}
	}

	if urlConf.Collection == "" {
		return urlPattern{}, errEmpty{"collection"}
	}

	hostRegex, err := regexp.Compile(urlConf.HostPattern)
	if err != nil {
		return urlPattern{}, fmt.Errorf("error: failed to compile hostpattern %q: %w",
			urlConf.HostPattern, err)
	}

	var pathRegex *regexp.Regexp

	if urlConf.PathPattern != "" {
		r, err := regexp.Compile(urlConf.PathPattern)
		if err != nil {
			return urlPattern{}, fmt.Errorf("error: failed to compile pathpattern %q: %w",
				urlConf.PathPattern, err)
		}

		pathRegex = r
	}

	return urlPattern{
		HostRegexp: hostRegex,
		PathRegexp: pathRegex,
		Collection: urlConf.Collection,
		FirstMsg:   urlConf.FirstMsg,
		Reactji:    urlConf.Reactji,
	}, nil
}

func (p *rareURLPattern) Configure(log *logrus.Logger, c *config.Config) error {
	p.log = log
	if c != nil {
		p.config = c.URLsConf

		var urlPatterns []urlPattern

		for _, urlConf := range c.URLsConf.URLs {
			urlPattern, err := newURLPattern(urlConf)
			if err != nil {
				return err
			}

			urlPatterns = append(urlPatterns, urlPattern)
		}

		p.urlPatterns = urlPatterns
	}

	return nil
}
