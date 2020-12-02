package models_test

import (
	"testing"

	"github.com/cpu/gorfbot/storage/models"
)

func TestTopic(t *testing.T) {
	topic := models.Topic{
		Creator: "U00000001",
		Channel: "C00000001",
		Topic:   ":tada: Test Topic!!!! :tada:",
		Date:    "1487690385.010607",
	}
	expectedStrForm :=
		`on 1487690385.010607 channel C00000001 was updated by U00000001 ` +
			`to have topic ":tada: Test Topic!!!! :tada:"`

	if topic.String() != expectedStrForm {
		t.Errorf("expected topic %#v to have String() %q, got %q",
			topic, expectedStrForm, topic.String())
	}
}
