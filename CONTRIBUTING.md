# Gorfbot Development Guide

Gorfbot is inspired by [Garfbot](https://github.com/doeg/garfbot) and so the
design is fairly similar.

Gorfbot was written recreationally in spare time without code review for a very
small, niche Slack. If parts of the code don't make sense you're probably right
and should clean it up or keep on trucking.

## Pre-Requisites 

* [Go 1.14+][go]

Optionally:
* GNU Make (for Makefile shortcuts)
  * In your friendly distro-specific package manager on Linux/BSD.
  * `brew install make` on MacOS 
  * ??? on Windows - sorry not sorry.
* [Delve][delve] (for debugging)

[go]: https://golang.org/doc/install
[delve]: https://github.com/go-delve/delve

## Setup

* `git clone https://github.com/cpu/gorfbot.git`
* `cd gorfbot`
* `make test`
* `make run`

## CI & Linting

All continuous integration is handled by [Github Actions][gh-actions], see
`.github/workflows/`.

All linting is handled by [golangci-lint][golangci-lint], see
[`.golangci-lint.yaml`][golangci-lint-config] for project specific
configuration. Use `//nolint:xxxx` annotations as required if linters
are being dumb.

[gh-actions]: https://docs.github.com/en/free-pro-team@latest/actions
[golangci-lint]: https://golangci-lint.run/
[golangci-lint-config]: https://github.com/cpu/gorfbot/blob/main/.golangci.yaml

## Unit Tests

Try to write them... Not all of the codebase has coverage but a good portion
does.

## Makefile

A very minimal `Makefile` is included, largely just to act as shell independent
aliases to speed up development.

Available targets:

* `make`, `make build` - builds `gorfbot` executable.
* `make run` - builds and runs `gorfbot` executable.
* `make debug` - builds and runs `gorfbot` executable in Delve debugger.
* `make clean` - deletes `gorfbot` executable from `make build`.
* `make lint` - run `golangci-lint`.
* `make test` - run unit tests (no race detector).
* `make test-race` - run unit tests (w/ race detector).
* `make test-cov` - run unit tests and open test coverage HTML in browser.
* `make snapshot` - make a [GoReleaser][GoReleaser] snapshot release build.

[GoReleaser]: https://goreleaser.com/quick-start/

## Extending the Bot

Bot commands are kept in separate packages under [`botcmd/`][botcmd-pkg], e.g.
[`botcmd/topics`][topics-pkg].

To extend the bot create a new package under `botcmd`, import
it from the [main bot code][bot.go], and have your command register itself from the
package's `init` function.

At a high level:

```bash
mkdir botcmd/mynewcmd
vim bot/bot.go # add `import _ "github.com/cpu/gorfbot/botcmd/mynewcmd"`
vim botcmd/mynewcmd/main.go # write cmd, register from init()
```

[botcmd-pkg]: https://github.com/cpu/gorfbot/tree/main/botcmd
[topics-pkg]: https://github.com/cpu/gorfbot/tree/main/botcmd/topics
[bot.go]: https://github.com/cpu/gorfbot/blob/main/bot/bot.go

### Importing New Command Packages

**!!! IMPORTANT !!**

Don't forget to update [`bot/bot.go`][bot.go] to add an empty import statement for your
new package! e.g.

```go
import (
...
  _ "github.com/cpu/gorfbot/botcmd/mynewcmd"
...
)
```

This is the magic that ensures your package's `init` function is
called to register the commands with the bot. It's a little bit awkward but
'works'. Improvements welcome!

**!!! IMPORTANT !!**

### Handlers

At a high level the handlers all receive a `botcmd.RunContext` instance that
gives them the `slack.Message` that caused the handler to be run. It also allows
access to a `slack.Client` for interacting with Slack and a `storage.Storage`
instance for finding/saving data.

Beyond manually interacting with the Slack API through the run context handlers
can also return a `botcmd.RunResult` with an optional message to post back to
the channel where the invoking message was, as well as zero or more reactji to
add to it.

### Configuration

Before any handler's are called there is a `Configure` function that is called
to provide a `logrus.Logger` instance set up with the correct loglevel based on
the command line flags provided to the bot. It also provides a `config.Config`
instance that the handler can use to configure itself based on the YAML config
the bot loaded (e.g. to set timeouts or API keys).

### Processing reactions...

For this you will want to register a new `botcmd.ReactionCommand`. See
[`botcmd/reactjiupdate/main.go`][reactjiupdate] for an example to copy.

[reactjiupdate]: https://github.com/cpu/gorfbot/blob/main/botcmd/reactjiupdate/main.go

### Processing messages matching a regular expression...

For this you will want to register a new `botcmd.PatternCommand`. See
[`botcmd/topicupdate/main.go`][topicupdate] for an example to copy.

[topicupdate]: https://github.com/cpu/gorfbot/blob/main/botcmd/topicupdate/main.go

### Adding a new command users can invoke...

For this you will want to register a new `botcmd.BasicCommand`. See
[`botcmd/hello/main.go`][hello] for a simple example to copy.

[hello]: https://github.com/cpu/gorfbot/blob/main/botcmd/hello/main.go

#### ... but I also want the command to have flags/arguments.

For this simply process the leftover message content passed to your
`botcmd.BasiCommand.Handler.Run` function. See [`botcmd/topics/main.go`][topics]
for an example to copy.

[topics]: https://github.com/cpu/gorfbot/blob/main/botcmd/topics/main.go

## Code Organization

Here is a rough overview of the codebase layout:

* [`cmd/gorfbot/`][gorfbot-pkg] -> the `gorfbot` command. Small driver program that
  loads config & instantiates all of the bot objects.

[gorfbot-pkg]: https://github.com/cpu/gorfbot/blob/main/cmd/gorfbot/main.go

* [`bot/`][bot-pkg] -> the bot logic for processing commands, dispatching slack events to
  pattern and reaction handlers as appropriate, etc.

[bot-pkg]: https://github.com/cpu/gorfbot/tree/main/bot

* [`slack/`][slack-pkg]  -> types and functions for interacting with Slack. To keep unit
  testing & future maintenance easy we keep everything behind an interface and
  try not to directly interact with `github.com/slack-go/slack` outside of this
  package. Very little of the overall Slack API surface is exposed through the
  bot's Slack interface (by design). If you need new events passed through
  you'll have to do some plumbing work. Gorfbot isn't a hyper generic bot
  building framework!

[slack-pkg]: https://github.com/cpu/gorfbot/tree/main/slack

* [`storage/storage.go`][storage-pkg] -> generic interface for bot data storage and
  retreival. Tries to abstract away the underlying DB technology as much as
  possible.

[storage-pkg]: https://github.com/cpu/gorfbot/tree/main/storage

* [`storage/models/`][models-pkg] -> model types used for storing/retreiving
  data, again as DB technology agnostic as possible. Minimal logic.

[models-pkg]: https://github.com/cpu/gorfbot/tree/main/storage/models

* [`storage/mongo/`][mongo-pkg] -> MongoDB specific implementation of the generic storage
  interface. Presently written to assume a hosted MongoDB Atlas instance but it
  should work with a local instance as well.

[mongo-pkg]: https://github.com/cpu/gorfbot/tree/main/storage/mongo

* [`config/`][config-pkg] -> configuration handling.

[config-pkg]: https://github.com/cpu/gorfbot/tree/main/config

* [`test/`][test-pkg] -> helper functions for tests.

[test-pkg]: https://github.com/cpu/gorfbot/tree/main/test

