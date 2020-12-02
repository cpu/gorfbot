//nolint:funlen
package config_test

import (
	"io/ioutil"
	"reflect"
	"testing"
	"time"

	"github.com/cpu/gorfbot/config"
)

func TestFromYAML(t *testing.T) {
	timeoutA := time.Second * 30
	timeoutB := time.Second * 35
	timeoutC := time.Second * 40
	oneHour := time.Hour
	testCases := []struct {
		name           string
		config         []byte
		expectedConfig *config.Config
	}{
		{
			name: "valid full YAML",
			config: []byte(`
MongoConf:
  Username: "user"
  Password: "pass"
  Hostname: "example.com"
  Database: "db"
  Options: "retryWrites=true&w=majority"
  ConnectTimeout: "30s"
  ReadTimeout: "35s"
  WriteTimeout: "40s"
SlackConf:
  APIToken: "token"
  Debug: true
  StateMaxAge: "1h"
FrogtipConf:
  UserAgent: "CoolFrog v1.0"
ReactjiKeysConf:
  Keywords:
    foo: 
      - bar
    baz: 
      - bop
      - pog
GISConf:
  CSEID: "xxx"
  APIKey: "yyy"
  Timeout: "30s"
URLsConf:
  URLs:
  - HostPattern: ".*\\.example.com"
    PathPattern: "\\/example\\/.*"
    Collection: "example_site_links"
    FirstMsg: ":sparkles: new example.com link shared :sparkles:"
    Reactji:
    - "link"
`),
			expectedConfig: &config.Config{
				MongoConf: config.MongoConfig{
					Username:       "user",
					Password:       "pass",
					Hostname:       "example.com",
					Database:       "db",
					Options:        "retryWrites=true&w=majority",
					ConnectTimeout: &timeoutA,
					ReadTimeout:    &timeoutB,
					WriteTimeout:   &timeoutC,
				},
				SlackConf: config.SlackConfig{
					APIToken:    "token",
					Debug:       true,
					StateMaxAge: &oneHour,
				},
				FrogtipConf: config.FrogtipConfig{
					UserAgent: "CoolFrog v1.0",
				},
				ReactjiKeysConf: config.ReactjiKeysConfig{
					Keywords: map[string][]string{
						"foo": {"bar"},
						"baz": {"bop", "pog"},
					},
				},
				GISConf: config.GISConfig{
					CSEID:   "xxx",
					APIKey:  "yyy",
					Timeout: &timeoutA,
				},
				URLsConf: config.URLsConfig{
					URLs: []config.URLConfig{
						{
							HostPattern: `.*\.example.com`,
							PathPattern: `\/example\/.*`,
							Collection:  "example_site_links",
							FirstMsg:    ":sparkles: new example.com link shared :sparkles:",
							Reactji:     []string{"link"},
						},
					},
				},
			},
		},
		{
			name: "valid partial YAML",
			config: []byte(`
MongoConf:
  Username: "user"
  Password: "pass"
  Hostname: "example.com"
  Database: "db"
SlackConf:
  APIToken: "token"
`),
			expectedConfig: &config.Config{
				MongoConf: config.MongoConfig{
					Username: "user",
					Password: "pass",
					Hostname: "example.com",
					Database: "db",
				},
				SlackConf: config.SlackConfig{
					APIToken: "token",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := config.FromYAML(tc.config)
			if err != nil {
				t.Fatalf("unexpected err reading config from YAML: %v", err)
			}
			if !reflect.DeepEqual(c, tc.expectedConfig) {
				t.Errorf("expected config %v got %v", tc.expectedConfig, c)
			}
		})
	}

	// Also try loading invalid YAML to ensure an error
	if _, err := config.FromYAML([]byte("a")); err == nil {
		t.Errorf("expected err loading invalid YAML, got nil")
	}
}

func TestFromYAMLFile(t *testing.T) {
	testConfig := []byte(`
MongoConf:
  Username: "user"
  Password: "pass"
  Hostname: "example.com"
  Database: "db"
SlackConf:
  APIToken: "token"
`)
	expectedConfig := &config.Config{
		MongoConf: config.MongoConfig{
			Username: "user",
			Password: "pass",
			Hostname: "example.com",
			Database: "db",
		},
		SlackConf: config.SlackConfig{
			APIToken: "token",
		},
	}

	f, err := ioutil.TempFile("", "test.config.*.yaml")
	if err != nil {
		t.Fatalf("failed to create tempfile for test YAML config")
	}

	if err := ioutil.WriteFile(f.Name(), testConfig, 0600); err != nil {
		t.Fatalf("failed to write test YAML config to %q: %v", f.Name(), err)
	} else if err := f.Close(); err != nil {
		t.Fatalf("failed to close test YAML config file: %v", err)
	}

	if c, err := config.FromYAMLFile(f.Name()); err != nil {
		t.Errorf("unexpected err loading YAML config from %q: %v", f.Name(), err)
	} else if !reflect.DeepEqual(c, expectedConfig) {
		t.Errorf("expected config %v got %v", expectedConfig, c)
	}

	// Also try loading invalid YAML file path to ensure an error
	if _, err := config.FromYAMLFile(f.Name() + "aaaa"); err == nil {
		t.Errorf("expected err loading invalid YAML config file path, got nil")
	}
}

func TestMongoConfig(t *testing.T) {
	errPrefix := "Mongo Config missing "
	testCases := []struct {
		name           string
		config         config.MongoConfig
		expectedErrMsg string
		expectedURI    string
	}{
		{
			name:           "Config Missing All",
			expectedErrMsg: errPrefix + "Username, Password, Hostname, Database",
		},
		{
			name: "Config Missing Username",
			config: config.MongoConfig{
				Password: "foo",
				Hostname: "foo",
				Database: "foo",
			},
			expectedErrMsg: errPrefix + "Username",
		},
		{
			name: "Config Missing Password",
			config: config.MongoConfig{
				Username: "foo",
				Hostname: "foo",
				Database: "foo",
			},
			expectedErrMsg: errPrefix + "Password",
		},
		{
			name: "Config Missing Hostname",
			config: config.MongoConfig{
				Username: "foo",
				Password: "foo",
				Database: "foo",
			},
			expectedErrMsg: errPrefix + "Hostname",
		},
		{
			name: "Config Missing Database",
			config: config.MongoConfig{
				Username: "foo",
				Password: "foo",
				Hostname: "foo",
			},
			expectedErrMsg: errPrefix + "Database",
		},
		{
			name: "No Options",
			config: config.MongoConfig{
				Username: "user",
				Password: "pass",
				Hostname: "host",
				Database: "db",
			},
			expectedURI: "mongodb+srv://user:pass@host/db",
		},
		{
			name: "With Options",
			config: config.MongoConfig{
				Username: "user",
				Password: "pass",
				Hostname: "host",
				Database: "db",
				Options:  "opts",
			},
			expectedURI: "mongodb+srv://user:pass@host/db?opts",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Check()
			if tc.expectedErrMsg != "" && err == nil {
				t.Errorf("expected err %q got nil", tc.expectedErrMsg)
			} else if tc.expectedErrMsg == "" && err != nil {
				t.Errorf("expected no err, got %v", err)
			} else if err != nil && err.Error() != tc.expectedErrMsg {
				t.Errorf("expected err %q got %q", tc.expectedErrMsg, err.Error())
			} else if tc.expectedErrMsg == "" && err == nil {
				if uri := tc.config.URI(); uri != tc.expectedURI {
					t.Errorf("expected URI %q got %q", tc.expectedURI, uri)
				}
			}
		})
	}
}

func TestSlackConfig(t *testing.T) {
	if err := (config.SlackConfig{}).Check(); err == nil {
		t.Errorf(
			"expected SlackConfig missing APIToken to return err from check, got nil")
	} else if err.Error() != "provided Slack Config missing APIToken" {
		t.Errorf(
			"expected SlackConfig missing APIToken to return err with correct msg, got %q",
			err.Error())
	}

	if err := (config.SlackConfig{APIToken: "a"}).Check(); err != nil {
		t.Errorf("unexpected SlackConfig valid err: %v\n", err)
	}
}
