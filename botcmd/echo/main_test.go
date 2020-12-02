package echo

import (
	"testing"

	"github.com/cpu/gorfbot/botcmd"
	"github.com/cpu/gorfbot/slack"
)

func TestRun(t *testing.T) {
	cmd := &echoCmd{}

	if _, err := cmd.Run("", botcmd.RunContext{}); err == nil {
		t.Errorf("expected err from Run('', nil) got nil")
	}

	ctx := botcmd.RunContext{Message: &slack.Message{}}
	if resp, err := cmd.Run("hello", ctx); err != nil {
		t.Errorf("unexpected err from Run: %v", err)
	} else if resp.Message != "hello" {
		t.Errorf("unexpected resp from Run: %v", resp)
	}
}

func TestConfigure(t *testing.T) {
	cmd := &echoCmd{}
	if err := cmd.Configure(nil, nil); err != nil {
		t.Errorf("expected no err from configure")
	}
}
