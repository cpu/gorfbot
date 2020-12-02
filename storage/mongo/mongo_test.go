package mongo

import (
	"testing"

	"github.com/cpu/gorfbot/config"
)

func TestNewMongoStorageNilConf(t *testing.T) {
	if _, err := NewMongoStorage(nil, nil); err == nil {
		t.Error("expected err from NewMongoStorage(nil), got nil")
	}
}

func TestNewMongoStorageInvalidConf(t *testing.T) {
	if _, err := NewMongoStorage(nil, &config.Config{}); err == nil {
		t.Error("expected err from NewMongoStorage w/ invalid config got nil")
	}
}
