//nolint:goerr113
package gis

import (
	"errors"
	"testing"

	"github.com/cpu/gorfbot/botcmd"
	"github.com/cpu/gorfbot/config"
	"github.com/cpu/gorfbot/test"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

func TestConfigure(t *testing.T) {
	log, _ := logtest.NewNullLogger()
	config := &config.Config{
		GISConf: config.GISConfig{
			APIKey: "xxx",
			CSEID:  "yyy",
		},
	}
	cmd := &gisCmd{}

	if err := cmd.Configure(log, config); err != nil {
		t.Errorf("expected no err from configure, got %v", err)
	} else if cmd.log != log {
		t.Errorf("expected log to be set to %p was %p", log, cmd.log)
	} else if cmd.config.APIKey != config.GISConf.APIKey {
		t.Errorf("expected API key to be set to %q was %q",
			config.GISConf.APIKey, cmd.config.APIKey)
	}

	cmd = &gisCmd{}
	if err := cmd.Configure(log, nil); err != nil {
		t.Errorf("expected no err from configure, got %v", err)
	} else if cmd.log != log {
		t.Errorf("expected log to be set to %p was %p", log, cmd.log)
	}
}

type mockGISAPI struct {
	t               *testing.T
	expectedConfig  config.GISConfig
	expectedOptions imageSearchOptions
	mockResults     []imageResult
	mockErr         error
}

func (m mockGISAPI) ImageSearch(conf config.GISConfig, opts imageSearchOptions) ([]imageResult, error) {
	if conf != m.expectedConfig {
		m.t.Errorf("ImageSearch expected config %v got %v", m.expectedConfig, conf)
	}

	if opts != m.expectedOptions {
		m.t.Errorf("ImageSearch expected search opts %#v got %#v",
			m.expectedOptions, opts)
	}

	return m.mockResults, m.mockErr
}

func TestSearchErr(t *testing.T) {
	log, _ := logtest.NewNullLogger()
	config := config.GISConfig{}
	cmd := &gisCmd{
		log: log,
		api: mockGISAPI{
			t:               t,
			expectedOptions: imageSearchOptions{Query: "test", Limit: 1},
			expectedConfig:  config,
			mockErr:         errors.New("search failed: google ran out of disk space"),
		},
	}
	expectedErr := `gis error search failed: google ran out of disk space`

	if _, err := cmd.Run("-random=false test", botcmd.RunContext{}); err == nil {
		t.Errorf("expected err from Run got nil")
	} else if actualErr := err.Error(); actualErr != expectedErr {
		t.Errorf("expected err %q got %q", expectedErr, actualErr)
	}
}

func TestSearchNoResults(t *testing.T) {
	log, logHook := logtest.NewNullLogger()
	config := config.GISConfig{
		APIKey: "aaa",
		CSEID:  "bbb",
	}
	cmd := &gisCmd{
		log:    log,
		config: config,
		api: mockGISAPI{
			t:               t,
			expectedOptions: imageSearchOptions{Query: "test", Limit: 10},
			expectedConfig:  config,
			mockResults:     []imageResult{},
		},
	}

	if res, err := cmd.Run("test", botcmd.RunContext{}); err != nil {
		t.Errorf("unexpected err from Run got %v", err)
	} else if res.Message != "" {
		t.Errorf("unexpected res message: %q", res.Message)
	} else if len(res.Reactji) != 1 || res.Reactji[0] != "zero" {
		t.Errorf("unexpected res reactji: %v", res.Reactji)
	}

	test.ExpectLastLog(t, logHook, logrus.InfoLevel, "gis got 0 search results")
}

func TestSearch(t *testing.T) {
	log, logHook := logtest.NewNullLogger()
	config := config.GISConfig{
		APIKey: "aaa",
		CSEID:  "bbb",
	}
	cmd := &gisCmd{
		log:    log,
		config: config,
		api: mockGISAPI{
			t:               t,
			expectedOptions: imageSearchOptions{Query: "test", Limit: 1},
			expectedConfig:  config,
			mockResults: []imageResult{
				{
					Title: "Big Fat Fake Data",
					URL:   "http://example.com/test.jpg",
				},
				{
					Title: "A Second Result",
					URL:   "http://example.com/test.2.jpg",
				},
			},
		},
	}

	expectedMsg := `:frame_with_picture: :mag: - _"Big Fat Fake Data"_
http://example.com/test.jpg

`

	if res, err := cmd.Run("-limit 1 -random=false test", botcmd.RunContext{}); err != nil {
		t.Errorf("unexpected err from Run got %v", err)
	} else if res.Message != expectedMsg {
		t.Errorf("expected res message %q got %q", expectedMsg, res.Message)
	}

	test.ExpectLastLog(t, logHook, logrus.InfoLevel, "gis got 2 search results")
}

func TestSearchOptions(t *testing.T) {
	log, logHook := logtest.NewNullLogger()
	config := config.GISConfig{
		APIKey: "aaa",
		CSEID:  "bbb",
	}
	cmd := &gisCmd{
		log:    log,
		config: config,
		api: mockGISAPI{
			t: t,
			expectedOptions: imageSearchOptions{
				Query:      "test one two",
				Limit:      2,
				Colour:     "blue",
				ColourType: "trans",
				Site:       "example.com",
				Size:       "small",
				Type:       "animated",
			},
			expectedConfig: config,
			mockResults: []imageResult{
				{
					Title: "Big Fat Fake Data",
					URL:   "http://example.com/test.jpg",
				},
				{
					Title: "A Second Result",
					URL:   "http://example.com/test.2.jpg",
				},
			},
		},
	}

	expectedMsg := `:frame_with_picture: :mag: - _"Big Fat Fake Data"_
http://example.com/test.jpg

:frame_with_picture: :mag: - _"A Second Result"_
http://example.com/test.2.jpg

`

	msg := "-limit 2 -random=false -color blue -colorType trans -site example.com -size small -type animated test one two"
	if res, err := cmd.Run(msg, botcmd.RunContext{}); err != nil {
		t.Errorf("unexpected err from Run got %v", err)
	} else if res.Message != expectedMsg {
		t.Errorf("expected res message %q got %q", expectedMsg, res.Message)
	}

	test.ExpectLastLog(t, logHook, logrus.InfoLevel, "gis got 2 search results")
}
