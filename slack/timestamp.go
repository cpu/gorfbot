package slack

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const expectedSlackTimestampComponents = 2

type errBadTimestamp struct {
	ts string
}

func (e errBadTimestamp) Error() string {
	return fmt.Sprintf("unable to convert Slack timestamp %q to time.Time", e.ts)
}

// Parses a Slack-style timestamp by removing the UUID component if it is
// present. Returns a time.Time instance or an err.
func (c clientImpl) ParseTimestamp(timestamp string) (time.Time, error) {
	components := strings.Split(timestamp, ".")
	if len(components) != expectedSlackTimestampComponents {
		c.log.Warnf("found weird timestamp %q, components %v", timestamp, components)
	}

	tsComponent := components[0]

	ts, err := strconv.ParseInt(tsComponent, 10, 64)
	if err != nil {
		return time.Time{}, errBadTimestamp{timestamp}
	}

	return time.Unix(ts, 0), nil
}
