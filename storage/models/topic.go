package models

import "fmt"

// Topic is a model for holding topic history information for a channel.
type Topic struct {
	// Creator is the ID of the user that changed the topic (note: not the
	// friendly username).
	Creator string
	// Channel is the ID of the channel that had its topic changed (note: not the
	// friendly channel name).
	Channel string
	// Topic is the topic that was set.
	Topic string
	// Date is the slack timestamp on which the topic date was set.
	Date string
}

// String is a simple debugging representation for the Topic model.
func (t Topic) String() string {
	return fmt.Sprintf("on %s channel %s was updated by %s to have topic %q",
		t.Date, t.Channel, t.Creator, t.Topic)
}
