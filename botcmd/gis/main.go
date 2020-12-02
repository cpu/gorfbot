package gis

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/cpu/gorfbot/botcmd"
	"github.com/cpu/gorfbot/config"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/customsearch/v1"
	"google.golang.org/api/option"
)

// # Helpful Documentation
//
// * REST Request API docs:
//     https://developers.google.com/custom-search/v1/reference/rest/v1/cse/list#request
// * REST Response API docs:
//     https://developers.google.com/custom-search/v1/reference/rest/v1/Search
// * Go Library:
//     https://github.com/googleapis/google-api-go-client/blob/master/customsearch/v1/customsearch-gen.go

const (
	cmdName         = "gis"
	maxQueryLimit   = int64(50) //nolint:gomnd
	maxDisplayLimit = int64(10) //nolint:gomnd
)

var (
	defaultTimeout                  = time.Second * 30
	linkRegexp                      = regexp.MustCompile(`<([^|]+)\|[^>]+>`)
	expectedLinkRegexpSubmatchCount = 2
)

type gisCmd struct {
	log    *logrus.Logger
	config config.GISConfig
	api    gisAPI
}

func init() {
	botcmd.MustAddCommand(&botcmd.BasicCommand{
		Name:        cmdName,
		Icon:        ":frame_with_picture:",
		Description: "Make a Google Image Search",
		Handler: &gisCmd{
			api: gisAPIImpl{},
		},
	})
}

type imageResult struct {
	Title string
	URL   string
}

type imageSearchOptions struct {
	Query      string
	Limit      int64
	ColourType string
	Colour     string
	Size       string
	Type       string
	Site       string
}

type errInvalidSize struct {
	size string
}

func (e errInvalidSize) Error() string {
	return fmt.Sprintf("Size %q is invalid", e.size)
}

type errInvalidType struct {
	typ string
}

func (e errInvalidType) Error() string {
	return fmt.Sprintf("Type %q is invalid", e.typ)
}

type errInvalidColour struct {
	colour string
}

func (e errInvalidColour) Error() string {
	return fmt.Sprintf("Color Type %q is invalid", e.colour)
}

type errInvalidLimit struct {
	msg string
}

func (e errInvalidLimit) Error() string {
	return e.msg
}

var errEmptyQuery = errors.New("provided Query is empty")

func (opts imageSearchOptions) valid() error {
	if opts.Query == "" {
		return errEmptyQuery
	}

	if opts.Limit < 1 {
		return errInvalidLimit{fmt.Sprintf("Limit %d is less than min, 1", opts.Limit)}
	}

	if opts.Limit > maxQueryLimit {
		return errInvalidLimit{fmt.Sprintf("Limit %d is greater than max, %d",
			opts.Limit, maxQueryLimit)}
	}

	validColourTypes := map[string]bool{
		"":      true,
		"color": true,
		"gray":  true,
		"mono":  true,
		"trans": true,
	}
	if !validColourTypes[opts.ColourType] {
		return errInvalidColour{opts.ColourType}
	}

	validTypes := map[string]bool{
		"":         true,
		"clipart":  true,
		"face":     true,
		"lineart":  true,
		"stock":    true,
		"photo":    true,
		"animated": true,
	}
	if !validTypes[opts.Type] {
		return errInvalidType{opts.Type}
	}

	validSizes := map[string]bool{
		"":        true,
		"huge":    true,
		"icon":    true,
		"large":   true,
		"medium":  true,
		"small":   true,
		"xlarge":  true,
		"xxlarge": true,
	}
	if !validSizes[opts.Size] {
		return errInvalidSize{opts.Size}
	}

	return nil
}

type gisAPI interface {
	ImageSearch(conf config.GISConfig, opts imageSearchOptions) ([]imageResult, error)
}

type gisAPIImpl struct{}

func (g gisAPIImpl) queryForOpts(
	svc *customsearch.Service,
	cseID string,
	opts imageSearchOptions) *customsearch.CseListCall {
	// Base request, always include cseID, search type == image, and the query
	req := svc.Cse.List().
		Cx(cseID).
		SearchType("image").
		Q(opts.Query)

	if opts.Limit != 0 {
		req = req.Num(opts.Limit)
	}

	// Optionally further customize the request based on the options
	if opts.ColourType != "" {
		req = req.ImgColorType(opts.ColourType)
	}

	if opts.Colour != "" {
		req = req.ImgDominantColor(opts.Colour)
	}

	if opts.Size != "" {
		req = req.ImgSize(opts.Size)
	}

	if opts.Type != "" {
		req = req.ImgType(opts.Type)
	}

	if opts.Site != "" {
		req = req.LinkSite(opts.Site)
	}

	return req
}

func (g gisAPIImpl) ImageSearch(conf config.GISConfig, opts imageSearchOptions) ([]imageResult, error) {
	if err := opts.valid(); err != nil {
		return nil, fmt.Errorf("invalid search options: %w", err)
	}

	timeout := defaultTimeout
	if conf.Timeout != nil {
		timeout = *conf.Timeout
	}

	ctx, cancel := config.ContextForTimeout(timeout)
	defer cancel()

	svc, err := customsearch.NewService(ctx, option.WithAPIKey(conf.APIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create customsearch client: %w", err)
	}

	req := g.queryForOpts(svc, conf.CSEID, opts)

	search, err := req.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to do search: %w", err)
	}

	results := make([]imageResult, len(search.Items))
	for i, res := range search.Items {
		results[i] = imageResult{
			Title: res.Title,
			URL:   res.Link,
		}
	}

	return results, nil
}

//nolint:funlen
func (cmd gisCmd) Run(text string, runCtx botcmd.RunContext) (botcmd.RunResult, error) {
	flagSet := flag.NewFlagSet(cmdName, flag.ContinueOnError)
	limitFlag := flagSet.Int64("limit", 1, fmt.Sprintf(
		"upper limit for number of images to return, max %d", maxDisplayLimit))
	randomFlag := flagSet.Bool("random", true, "choose images randomly, or in order")
	colourTypeFlag := flagSet.String("colorType", "", "[color|gray|mono|trans]")
	colourFlag := flagSet.String("color", "", "[black|blue|etc]")
	sizeFlag := flagSet.String("size", "", "[huge|icon|large|medium|smal|xlarge|xxlarge]")
	typeFlag := flagSet.String("type", "", "[clipart|face|lineart|stock|photo|animated]")
	siteFlag := flagSet.String("site", "", "URL that must be linked to by all result sites")

	if respText := botcmd.ParseFlags(text, flagSet); respText != "" {
		return botcmd.RunResult{Message: respText}, nil
	}

	if *limitFlag > maxDisplayLimit {
		return botcmd.RunResult{
			Message: fmt.Sprintf("-limit %d is greater than max, %d",
				*limitFlag, maxDisplayLimit),
		}, nil
	}

	rest := strings.Join(flagSet.Args(), " ")

	// Work around the way that Slack turns links into their own format in messages.
	site := *siteFlag
	if site != "" {
		linkMatches := linkRegexp.FindStringSubmatch(site)
		if len(linkMatches) == expectedLinkRegexpSubmatchCount {
			site = linkMatches[1]
		}
	}

	queryLimit := *limitFlag

	// Always use a larger limit in the search when random flag is enabled
	// We truncate the displayed records to the chosen result.
	if *randomFlag {
		queryLimit = 10
	}

	opts := imageSearchOptions{
		Query:      rest,
		Limit:      queryLimit,
		ColourType: *colourTypeFlag,
		Colour:     *colourFlag,
		Size:       *sizeFlag,
		Type:       *typeFlag,
		Site:       site,
	}

	cmd.log.Infof("% searching with options %#v", cmdName, opts)

	results, err := cmd.api.ImageSearch(cmd.config, opts)
	if err != nil {
		return botcmd.RunResult{}, fmt.Errorf("%s error %w", cmdName, err)
	}

	numResults := int64(len(results))

	cmd.log.Infof("%s got %d search results", cmdName, numResults)

	if numResults < 1 {
		return botcmd.RunResult{
			Reactji: []string{"zero"},
		}, nil
	}

	limit := numResults
	if *limitFlag > 0 && *limitFlag < numResults {
		limit = *limitFlag
	}

	if *randomFlag {
		cmd.log.Tracef("%s results pre-random: %#v\n", cmdName, results)
		rand.Shuffle(len(results), func(i, j int) { results[i], results[j] = results[j], results[i] })
		cmd.log.Tracef("%s post random: %#v\n", cmdName, results)
	}

	buf := new(bytes.Buffer)

	for i := int64(0); i < limit; i++ {
		res := results[i]
		fmt.Fprintf(buf, ":frame_with_picture: :mag: - _%q_\n%s\n\n", res.Title, res.URL)
	}

	return botcmd.RunResult{Message: buf.String()}, nil
}

func (cmd *gisCmd) Configure(log *logrus.Logger, c *config.Config) error {
	cmd.log = log
	if c != nil {
		cmd.config = c.GISConf

		if c.GISConf.RandomSeed > 0 {
			rand.Seed(c.GISConf.RandomSeed)
		} else {
			rand.Seed(time.Now().UnixNano())
		}
	}

	return nil
}
