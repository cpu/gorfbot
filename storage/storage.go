package storage

import "github.com/cpu/gorfbot/storage/models"

// FindOptions is a struct for options common to most find operations: limiting
// result counts, sorting by a field name, and indicating if the sort is
// ascending or descending order.
type FindOptions struct {
	// Limit on maximum number of results to return.
	Limit int64
	// SortField is the name of a collection field to sort by.
	SortField string
	// Asc indicates if the sort is ascending or descending (default).
	Asc bool
}

// GetTopicOptions is a struct for customizing GetTopics.
type GetTopicOptions struct {
	FindOptions
	// Channel ID of the channel to retrieve topics for (note: an ID like
	// 'C123456' not a friendly name like '#general').
	Channel string
}

// GetEmojiOptions is a struct for customizing GetEmoji.
type GetEmojiOptions struct {
	FindOptions
	// User ID of the user to retrieve emoji history for (note: an ID like 'U1234'
	// not a friendly name like '@daniel').
	User string
	// Emoji name to retrieve emoji history for.
	Emoji string
	// Reaction indicates if the returned emoji info should be for reactions, or
	// normal emoji usage in messages (default).
	Reaction bool
}

// GetThemeOptions is a struct for customizing GetThemes.
type GetThemeOptions struct {
	FindOptions
	// User is a User ID that if provided will be used to limit results to just themes
	// saved by that user ID.
	User string
	// Name is a name of a theme and if provided will be used to limit results to
	// just themes matching that name.
	Name string
}

//go:generate mockgen -destination=mocks/mock_storage.go -package=mocks . Storage
// Storage is an interface describing all of the operations a Gorfbot storage
// backend must provide.
type Storage interface {
	// GetTopics returns topic models matching the options criteria.
	GetTopics(opts GetTopicOptions) ([]models.Topic, error)
	// AddTopic adds a topic model to the storage.
	AddTopic(topic models.Topic) error

	// GetEmoji returns emojis models matching the options criteria.
	GetEmoji(opts GetEmojiOptions) ([]models.Emoji, error)

	// UpsertEmojiCount upserts the provided emoji model, either increasing or
	// decreasing the count based on the decrement parameter (default: increment).
	// It returns the updated model.
	UpsertEmojiCount(emoji models.Emoji, decrement bool) (models.Emoji, error)

	// UsertURLCount upserts the provided url model in the provided collection
	// name. It returns the updated model.
	UpsertURLCount(collection string, urlCount models.URLCount) (models.URLCount, error)

	// GetThemes returns theme models matching the options criteria.
	GetThemes(opts GetThemeOptions) ([]models.Theme, error)
	// AddTheme adds a theme model to the storage.
	AddTheme(theme models.Theme) error
}
