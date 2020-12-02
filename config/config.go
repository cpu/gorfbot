package config

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// ContextForTimeout returns a context with the given timeout applied to an
// empty background context.
func ContextForTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// Config is a structure describing the overall gorfbot configuration.
type Config struct {
	MongoConf       MongoConfig       `yaml:"MongoConf"`
	SlackConf       SlackConfig       `yaml:"SlackConf"`
	ReactjiKeysConf ReactjiKeysConfig `yaml:"ReactjiKeysConf"`
	FrogtipConf     FrogtipConfig     `yaml:"FrogtipConf"`
	GISConf         GISConfig         `yaml:"GISConf"`
	URLsConf        URLsConfig        `yaml:"URLsConf"`
	MkthemeConf     MkthemeConfig     `yaml:"MkthemeConf"`
}

var ErrNilConfig = errors.New("config was nil")

// FromYAML constructs a Config instance from the given serialized Config YAML
// bytes or returns an error.
func FromYAML(configBytes []byte) (*Config, error) {
	var c Config
	if err := yaml.Unmarshal(configBytes, &c); err != nil {
		return nil, fmt.Errorf("YAML unmarshaling err: %w", err)
	}

	return &c, nil
}

// FromYAMLFile constructs a Config instance from the path to a YAML serialized
// Config file or returns an error.
func FromYAMLFile(configFilePath string) (*Config, error) {
	configBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("YAML config file processing err: %w", err)
	}

	return FromYAML(configBytes)
}

// MongoConfig describes configuration required to connect to a MongoDB instance.
type MongoConfig struct {
	// Username - required
	Username string `yaml:"Username"`
	// Password - required
	Password string `yaml:"Password"`
	// Hostname - required
	Hostname string `yaml:"Hostname"`
	// Database - required
	Database string `yaml:"Database"`
	// Options  - may be omitted. Everything after the `?` in a connection URI.
	Options string `yaml:"Options"`
	// ConnectTimeout - may be omitted.
	ConnectTimeout *time.Duration `yaml:"ConnectTimeout"`
	// ReadTimeout - may be omitted.
	ReadTimeout *time.Duration `yaml:"ReadTimeout"`
	// WriteTimeout - may be omitted.
	WriteTimeout *time.Duration `yaml:"WriteTimeout"`
}

type errMongoMissingConfig struct {
	what []string
}

func (e errMongoMissingConfig) Error() string {
	return fmt.Sprintf("Mongo Config missing %s", strings.Join(e.what, ", "))
}

// Check verifies a MongoConfig is valid. It returns an error if there are missing
// field values.
func (c MongoConfig) Check() error {
	var missing []string

	if c.Username == "" {
		missing = append(missing, "Username")
	}

	if c.Password == "" {
		missing = append(missing, "Password")
	}

	if c.Hostname == "" {
		missing = append(missing, "Hostname")
	}

	if c.Database == "" {
		missing = append(missing, "Database")
	}

	if len(missing) > 0 {
		return errMongoMissingConfig{missing}
	}

	return nil
}

// URI returns the Mongo connection URI for the given config.
func (c MongoConfig) URI() string {
	var options string
	if c.Options != "" {
		options = "?" + c.Options
	}

	return fmt.Sprintf(
		"mongodb+srv://%s:%s@%s/%s%s",
		c.Username, c.Password, c.Hostname, c.Database, options,
	)
}

// SlackConfig describes configuration required to connect to a Slack instance.
type SlackConfig struct {
	// APIToken for authenticating to Slack - required.
	APIToken string `yaml:"APIToken"`
	// Debug enables/disables the Slack API client debug option - optional. Debug messages
	// will be sent to the `Info` level of the bot's logger.
	Debug bool `yaml:"Debug"`
	// How long to wait before refreshing slack state (channels, users, etc).
	StateMaxAge *time.Duration `yaml:"StateMaxAge"`
}

var errMissingSlackAPIToken = errors.New("provided Slack Config missing APIToken")

// Check verifies a SlackConfig is valid. It returns an error if there are
// missing field values.
func (c SlackConfig) Check() error {
	if c.APIToken == "" {
		return errMissingSlackAPIToken
	}

	return nil
}

// ReactjiKeysConfig describes a mapping of keywords to lists of reactions to apply
// when the keyword is seen.
type ReactjiKeysConfig struct {
	// Keywords is a map of keyword to reaction list.
	Keywords map[string][]string `yaml:"Keywords"`
}

// FrogtipConfig describes configuration used by the Frogtips botcmd.
type FrogtipConfig struct {
	// UserAgent of the HTTP client for the Frogtips API.
	UserAgent string `yaml:"UserAgent"`
}

// GISConfig describes configuration used by the Google Image Search botcmd.
type GISConfig struct {
	// Timeout for searches.
	Timeout *time.Duration `yaml:"Timeout"`
	// Google CSEID for API access.
	CSEID string `yaml:"CSEID"`
	// Google APIKey for API access.
	APIKey string `yaml:"APIKey"`
	// RandomSeed for randomizing image results.
	RandomSeed int64 `yaml:"RandomSeed"`
}

// URLsConfig is a list of URLConfigs.
type URLsConfig struct {
	// URLs is a list of URLConfig.
	URLs []URLConfig `yaml:"URLs"`
}

// URLConfig describes a host (and optional path) pattern for matching URLs and
// tracking occurrences with special logic for never before seen URLs.
type URLConfig struct {
	// HostPattern is a regex that must match on the URL's host component.
	HostPattern string `yaml:"HostPattern"`
	// PathPattern is an optional regex that must match on the URL's path component.
	PathPattern string `yaml:"PathPattern"`
	// Collection is the name of a storage collection for the occurrences of this
	// URLConfig to be tracked in.
	Collection string `yaml:"Collection"`
	// FirstMsg is a message to post in response to matches of this URLConfig that
	// have never been seen before.
	FirstMsg string `yaml:"FirstMsg"`
	// Reactji is a list of reactions (no ":" delimiters) to post in response to
	// matches of this URLConfig.
	Reactji []string `yaml:"Reactji"`
}

// Mkthemeconfig describes configuration used by the mktheme botcmd.
type MkthemeConfig struct {
	// RandomSeed for seeding the colour generator.
	RandomSeed int64 `yaml:"RandomSeed"`
}
