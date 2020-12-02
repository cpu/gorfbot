//nolint:goerr113
package frogtip

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"

	"github.com/cpu/gorfbot/botcmd"
	"github.com/cpu/gorfbot/config"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

type mockFrogClient struct {
	t                 *testing.T
	expectedUserAgent string
	response          *http.Response
	err               error
}

func (m *mockFrogClient) Do(req *http.Request) (*http.Response, error) {
	if actualUA := req.Header.Get("User-Agent"); actualUA != m.expectedUserAgent {
		m.t.Errorf("expected UA %q got %q", m.expectedUserAgent, actualUA)
	}

	return m.response, m.err
}

func setup(
	t *testing.T,
	apiUserAgent string,
	expectedUserAgent string,
	mockResponse *http.Response,
	mockErr error) *frogtipCmd {
	log, _ := logtest.NewNullLogger()

	return &frogtipCmd{
		log: log,
		api: frogAPI{
			userAgent: apiUserAgent,
			client: &mockFrogClient{
				t:                 t,
				expectedUserAgent: expectedUserAgent,
				response:          mockResponse,
				err:               mockErr,
			},
		},
	}
}

func TestRunAPIErr(t *testing.T) {
	// NB: Test with a Custom UA, expect to see that UA.
	cmd := setup(t, "Custom UA", "Custom UA", nil, errors.New("lizard tips only"))

	expectedErrMsg := `frogtip cmd error making get tips request: lizard tips only`

	if _, err := cmd.Run("", botcmd.RunContext{}); err == nil {
		t.Errorf("expected err from run, got nil")
	} else if actualErrMsg := err.Error(); actualErrMsg != expectedErrMsg {
		t.Errorf("expected err %q got %q", expectedErrMsg, actualErrMsg)
	}
}

func TestRunAPIBadStatus(t *testing.T) {
	mockBody := ioutil.NopCloser(bytes.NewReader([]byte{}))
	mockResponse := &http.Response{
		StatusCode: 420,
		Body:       mockBody,
	}
	// NB: Test with empty UA, expect to see default UA.
	cmd := setup(t, "", defaultUserAgent, mockResponse, nil)
	expectedErrMsg := `frogtip cmd got non-200 response from tips API: 420`

	if _, err := cmd.Run("", botcmd.RunContext{}); err == nil {
		t.Errorf("expected err from run, got nil")
	} else if actualErrMsg := err.Error(); actualErrMsg != expectedErrMsg {
		t.Errorf("expected err %q got %q", expectedErrMsg, actualErrMsg)
	}
}

type errReader struct {
	err error
}

func (r errReader) Read(p []byte) (int, error) {
	return 0, r.err
}

func (r errReader) Close() error {
	return nil
}

func TestRunAPIBadBodyRead(t *testing.T) {
	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       errReader{err: errors.New("bad read, sorry m8")},
	}
	cmd := setup(t, "", defaultUserAgent, mockResponse, nil)
	expectedErrMsg := `frogtip cmd got err reading tips response: bad read, sorry m8`

	if _, err := cmd.Run("", botcmd.RunContext{}); err == nil {
		t.Errorf("expected err from run w/ err API, got nil")
	} else if actualErrMsg := err.Error(); actualErrMsg != expectedErrMsg {
		t.Errorf("expected err %q got %q", expectedErrMsg, actualErrMsg)
	}
}

func TestRunAPIBadJSON(t *testing.T) {
	badJSON := []byte("{")
	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewReader(badJSON)),
	}
	cmd := setup(t, "", defaultUserAgent, mockResponse, nil)
	expectedErrMsg := `frogtip cmd hit err unmarshaling tips response: unexpected end of JSON input`

	if _, err := cmd.Run("", botcmd.RunContext{}); err == nil {
		t.Errorf("expected err from run, got nil")
	} else if actualErrMsg := err.Error(); actualErrMsg != expectedErrMsg {
		t.Errorf("expected err %q got %q", expectedErrMsg, actualErrMsg)
	}
}

func TestRunAPINoTips(t *testing.T) {
	noTipsJSON, _ := json.Marshal(&tipsResult{})
	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewReader(noTipsJSON)),
	}
	cmd := setup(t, "", defaultUserAgent, mockResponse, nil)
	expectedErrMsg := `frogtip cmd tip API had no tips :-(`

	if _, err := cmd.Run("", botcmd.RunContext{}); err == nil {
		t.Errorf("expected err from run, got nil")
	} else if actualErrMsg := err.Error(); actualErrMsg != expectedErrMsg {
		t.Errorf("expected err %q got %q", expectedErrMsg, actualErrMsg)
	}
}

func TestRun(t *testing.T) {
	tipsJSON, _ := json.Marshal(&tipsResult{
		Tips: []tip{
			{
				Tip:    `frogz \"rule\"`,
				Number: 99,
			},
		},
	})
	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewReader(tipsJSON)),
	}
	customUA := "Secret Frog!"
	cmd := setup(t, customUA, customUA, mockResponse, nil)
	expectedMessage := `:frog: :speech_balloon: "frogz \"rule\""`
	expectedReactions := []string{"yin_yang", "pray"}

	if res, err := cmd.Run("", botcmd.RunContext{}); err != nil {
		t.Errorf("unexpected err from run: %v", err)
	} else if res.Message != expectedMessage {
		t.Errorf("expected msg %q got %q", expectedMessage, res.Message)
	} else if !reflect.DeepEqual(res.Reactji, expectedReactions) {
		t.Errorf("expected reactions %v got %v", expectedReactions, res.Reactji)
	}
}

func TestConfigure(t *testing.T) {
	log, _ := logtest.NewNullLogger()
	config := &config.Config{
		FrogtipConf: config.FrogtipConfig{
			UserAgent: "Secret Frog v1.0",
		},
	}
	cmd := &frogtipCmd{}

	if err := cmd.Configure(log, config); err != nil {
		t.Errorf("expected no err from configure, got %v", err)
	} else if cmd.log != log {
		t.Errorf("expected log to be set to %p was %p", log, cmd.log)
	} else if cmd.api.userAgent != config.FrogtipConf.UserAgent {
		t.Errorf("expected useragent to be set to %q was %q",
			config.FrogtipConf.UserAgent, cmd.api.userAgent)
	}

	// Loading a nil config shouldn't err either
	cmd = &frogtipCmd{}

	if err := cmd.Configure(log, nil); err != nil {
		t.Errorf("expected no err from configure, got %v", err)
	} else if cmd.log != log {
		t.Errorf("expected log to be set to %p was %p", log, cmd.log)
	} else if cmd.api.userAgent != "" {
		t.Errorf("expected useragent to be set to empty was %q", cmd.api.userAgent)
	}
}
