package main

import (
	"flag"
	"strings"

	"github.com/cpu/gorfbot/bot"
	"github.com/cpu/gorfbot/config"
	"github.com/sirupsen/logrus"
)

var (
	configPath = flag.String(
		"config", "./config.yml", "Path to YAML configuration file")

	logLevel = flag.String(
		"loglevel", "WARN", "Log msgs only at levels >= the provided logLevel")
)

func onErrQuit(log *logrus.Logger, e error) {
	if e != nil {
		log.Fatalf("Gorfbot fatal error: %v", e.Error())
	}
}

func stringToLevel(levelStr string) logrus.Level {
	switch strings.ToLower(levelStr) {
	case "error":
		return logrus.ErrorLevel
	case "warn":
		return logrus.WarnLevel
	case "info":
		return logrus.InfoLevel
	case "debug":
		return logrus.DebugLevel
	case "trace":
		return logrus.TraceLevel
	}

	logrus.Warnf(`Unknown log level: %q Using "warn"`, levelStr)

	return logrus.WarnLevel
}

func main() {
	flag.Parse()

	// Create a logger.
	var log = logrus.New()

	log.SetLevel(stringToLevel(*logLevel))
	log.Info("Welcome to Gorfbot")

	// Read a Config from YAML.
	c, err := config.FromYAMLFile(*configPath)
	onErrQuit(log, err)
	log.Infof("Read config from %q", "config.yml")

	// Create a Bot instance from the config.
	garf, err := bot.New(log, c)
	onErrQuit(log, err)
	log.Info("Starting bot loop")

	// Run the Bot. This will never return.
	garf.Run()
}
