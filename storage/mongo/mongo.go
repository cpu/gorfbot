package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cpu/gorfbot/config"
	"github.com/cpu/gorfbot/storage"
	"github.com/cpu/gorfbot/storage/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// mongoStorage is the implementation of the Storage interface for Mongo DB.
type mongoStorage struct {
	log    *logrus.Logger
	config config.MongoConfig
	client *mongo.Client
}

var defaultTimeout = time.Second * 60

// NewMongoStorage returns a Storage implementation for the given config backed by
// MongoDB. The config.MongoConf should be populated with a connection URI. Before
// returning a connection to the MongoDB instance is made and a Ping operation
// performed to verify the connection.
func NewMongoStorage(log *logrus.Logger, c *config.Config) (storage.Storage, error) {
	if log == nil {
		log = logrus.New()
	}

	if c == nil {
		return nil, fmt.Errorf("mongo storage err: %w", config.ErrNilConfig)
	}

	if err := c.MongoConf.Check(); err != nil {
		return nil, fmt.Errorf("mongo storage config err: %w", err)
	}

	connectTimeout := defaultTimeout
	if c.MongoConf.ConnectTimeout != nil {
		connectTimeout = *c.MongoConf.ConnectTimeout
	}

	ctx, _ := config.ContextForTimeout(connectTimeout)

	clientOpts := options.Client().ApplyURI(c.MongoConf.URI())

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("mongo client connect err: %w", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("mono client ping err: %w", err)
	}

	return mongoStorage{
		log:    log,
		config: c.MongoConf,
		client: client,
	}, nil
}

// findOptions creates Mongo FindOptions from the gorfbot specific
// storage.FindOptions. It applies sort options and limit.
func findOptions(opts storage.FindOptions) *options.FindOptions {
	mongoOpts := options.Find()

	sortValue := -1
	if opts.Asc {
		sortValue = 1
	}

	mongoOpts.SetSort(bson.D{bson.E{Key: opts.SortField, Value: sortValue}})

	limit := int64(0)
	if opts.Limit > 0 {
		limit = opts.Limit
	}

	mongoOpts.SetLimit(limit)

	return mongoOpts
}

// readCtx creates a context for reading based on the configured mongo read timeout,
// or the default read timeout. Note: this is rooted with a background context
// presently and can't be used as a nested context.
func (m mongoStorage) readCtx() context.Context {
	readTimeout := defaultTimeout
	if m.config.ReadTimeout != nil {
		readTimeout = *m.config.ReadTimeout
	}

	ctx, _ := config.ContextForTimeout(readTimeout)

	return ctx
}

// return the collection for topics.
func (m mongoStorage) topicsCollection() *mongo.Collection {
	return m.collection("topics")
}

// GetTopics reads Topic models from the mongo topics collection.
func (m mongoStorage) GetTopics(opts storage.GetTopicOptions) ([]models.Topic, error) {
	ctx := m.readCtx()
	collection := m.topicsCollection()

	var filter interface{}
	if opts.Channel != "" {
		filter = bson.M{"channel": opts.Channel}
	} else {
		filter = bson.D{}
	}

	findOpts := findOptions(opts.FindOptions)

	cursor, err := collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("mongo client topics collection find err: %w", err)
	}
	defer cursor.Close(ctx)

	var results []models.Topic

	for cursor.Next(ctx) {
		var topic models.Topic

		err := cursor.Decode(&topic)
		if err != nil {
			return nil, fmt.Errorf("mongo client topic decode err: %w", err)
		}

		results = append(results, topic)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("mongo client topic cursor err: %w", err)
	}

	return results, nil
}

// GetEmoji reads Emoji models from the mongo emoji collection, or reactji
// collection, as appropriate.
func (m mongoStorage) GetEmoji(opts storage.GetEmojiOptions) ([]models.Emoji, error) {
	ctx := m.readCtx()

	// Default to finding data from the emoji collection for emoji in messages.
	collection := m.emojiCollection()
	// But if the reactji is requested in the opts, use the reaction collection
	// instead (legacy reasons)...
	if opts.Reaction {
		collection = m.reactionCollection()
	}

	filter := bson.D{}
	if opts.User != "" {
		filter = append(filter, bson.E{Key: "user", Value: opts.User})
	}

	if opts.Emoji != "" {
		filter = append(filter, bson.E{Key: "emoji", Value: opts.Emoji})
	}

	findOpts := findOptions(opts.FindOptions)

	cursor, err := collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("mongo client emoji collection find err: %w", err)
	}
	defer cursor.Close(ctx)

	var results []models.Emoji

	for cursor.Next(ctx) {
		var emoji models.Emoji
		if err := cursor.Decode(&emoji); err != nil {
			return nil, fmt.Errorf("mongo client emoji decode err: %w", err)
		}

		// Because legacy data doesn't store the reaction field set it to match the
		// lookup opts.
		emoji.Reaction = opts.Reaction
		results = append(results, emoji)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("mongo client emoji cursor err: %w", err)
	}

	return results, nil
}

// writeCtx creates a context for writing based on the configured mongo write
// timeout, or the default write timeout. Note: like readCtx() this is rooted with
// a background context presently and can't be used as a nested context.
func (m mongoStorage) writeCtx() context.Context {
	writeTimeout := defaultTimeout
	if m.config.WriteTimeout != nil {
		writeTimeout = *m.config.WriteTimeout
	}

	ctx, _ := config.ContextForTimeout(writeTimeout)

	return ctx
}

// AddTopic adds a topic model to the topics collection.
func (m mongoStorage) AddTopic(topic models.Topic) error {
	ctx := m.writeCtx()
	collection := m.topicsCollection()

	_, err := collection.InsertOne(ctx, topic)
	if err != nil {
		return fmt.Errorf("mongo client topic add err: %w", err)
	}

	return nil
}

// collection returns a mongo collection for the provided name.
func (m mongoStorage) collection(name string) *mongo.Collection {
	db := m.client.Database(m.config.Database)
	return db.Collection(name)
}

// emojiCollection returns the collection for emoji counts.
func (m mongoStorage) emojiCollection() *mongo.Collection {
	return m.collection("panoptimojis")
}

// reactionCollection returns the collection for reaction emoji counts.
func (m mongoStorage) reactionCollection() *mongo.Collection {
	return m.collection("panoptireactjis")
}

// UpsertEmojiCount updates an emoji or reaction model's count to increase or
// decrease it depending on the decrement argument. By default the usage count
// is incremented.
func (m mongoStorage) UpsertEmojiCount(emoji models.Emoji, decrement bool) (models.Emoji, error) {
	ctx := m.writeCtx()

	// Default to inserting/incrementing in the emoji collection for emoji in messages.
	collection := m.emojiCollection()
	// But if the emoji is a reaction, use the reaction collection instead (legacy
	// reasons)...
	if emoji.Reaction {
		collection = m.reactionCollection()
	}

	// Filter by user/emoji
	filter := bson.D{
		bson.E{Key: "user", Value: emoji.User},
		bson.E{Key: "emoji", Value: emoji.Emoji},
	}

	// Increment existing counts if found.
	updateCount := 1
	if decrement {
		// Unless we're decrementing.
		updateCount = -1
	}

	update := bson.D{bson.E{
		Key:   "$inc",
		Value: bson.M{"count": updateCount},
	}}

	// Upsert to add if not exists
	opts := options.FindOneAndUpdate().SetUpsert(true)

	var updatedEmoji models.Emoji

	err := collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updatedEmoji)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return models.Emoji{}, fmt.Errorf("mongo upsert emoji count failure: %w", err)
	} else if errors.Is(err, mongo.ErrNoDocuments) {
		emoji.Count = 0
		return emoji, nil
	}

	// Because legacy data doesn't store the reaction field set it to match the lookup.
	updatedEmoji.Reaction = emoji.Reaction

	return updatedEmoji, nil
}

type errNoSuchCollection struct {
	name string
}

func (e errNoSuchCollection) Error() string {
	return fmt.Sprintf("no such collection: %q", e.name)
}

// UpsertURLCount updates a URLCount model's occurrence count.
func (m mongoStorage) UpsertURLCount(collectionName string, urlCount models.URLCount) (models.URLCount, error) {
	ctx := m.writeCtx()

	collection := m.collection(collectionName)
	if collection == nil {
		return models.URLCount{}, errNoSuchCollection{collectionName}
	}

	// Filter by URL
	filter := bson.D{
		bson.E{Key: "url", Value: urlCount.URL},
	}
	// Increment occurrences if found
	update := bson.D{bson.E{
		Key:   "$inc",
		Value: bson.M{"occurrences": 1},
	}}
	// Upsert to add if not exists
	opts := options.FindOneAndUpdate().SetUpsert(true)

	var updatedCount models.URLCount

	err := collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updatedCount)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return models.URLCount{}, fmt.Errorf("mongo upsert URL count failure: %w", err)
	} else if errors.Is(err, mongo.ErrNoDocuments) {
		urlCount.Occurrences = 0
		return urlCount, nil
	}

	return updatedCount, nil
}

// themeCollection returns the mongo collection for themes.
func (m mongoStorage) themeCollection() *mongo.Collection {
	return m.collection("themes")
}

// GetThemes returns theme models from the theme collection.
func (m mongoStorage) GetThemes(opts storage.GetThemeOptions) ([]models.Theme, error) {
	ctx := m.readCtx()

	collection := m.themeCollection()

	filter := bson.D{}
	if opts.User != "" {
		filter = append(filter, bson.E{Key: "user", Value: opts.User})
	}

	if opts.Name != "" {
		filter = append(filter, bson.E{Key: "name", Value: opts.Name})
	}

	findOpts := findOptions(opts.FindOptions)

	cursor, err := collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("mongo client theme collection find err: %w", err)
	}
	defer cursor.Close(ctx)

	var results []models.Theme

	for cursor.Next(ctx) {
		var theme models.Theme
		if err := cursor.Decode(&theme); err != nil {
			return nil, fmt.Errorf("mongo client theme decode err: %w", err)
		}

		results = append(results, theme)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("mongo client theme cursor err: %w", err)
	}

	return results, nil
}

// AddTheme adds a theme to the theme collection.
func (m mongoStorage) AddTheme(theme models.Theme) error {
	ctx := m.writeCtx()
	collection := m.themeCollection()

	_, err := collection.InsertOne(ctx, theme)
	if err != nil {
		return fmt.Errorf("mongo client theme add err: %w", err)
	}

	return nil
}
