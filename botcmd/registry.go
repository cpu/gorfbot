package botcmd

import "fmt"

// CommandRegistry describes a collection of basic commands, pattern commands
// and reaction handlers. It is not thread safe - do not use concurrently.
type CommandRegistry struct {
	cmdsList []*BasicCommand
	cmdsMap  map[string]*BasicCommand

	patternsList []*PatternCommand
	patternsMap  map[string]*PatternCommand

	reactionHandlerList []*ReactionCommand
	reactionHandlerMap  map[string]*ReactionCommand
}

// NewRegistry constructs a new CommandRegistry.
func NewRegistry() *CommandRegistry {
	return &CommandRegistry{
		cmdsMap:            make(map[string]*BasicCommand),
		patternsMap:        make(map[string]*PatternCommand),
		reactionHandlerMap: make(map[string]*ReactionCommand),
	}
}

// GetConfigurables returns a list of all of the configurables in the registry.
// This includes basic commands, patterns and reaction handlers.
func (c CommandRegistry) GetConfigurables() []Configurable {
	var allConfigurables []Configurable //nolint:prealloc

	for _, c := range c.GetCommands() {
		allConfigurables = append(allConfigurables, c.Handler)
	}

	for _, p := range c.GetPatterns() {
		allConfigurables = append(allConfigurables, p.Handler)
	}

	for _, h := range c.GetReactionHandlers() {
		allConfigurables = append(allConfigurables, h.Handler)
	}

	return allConfigurables
}

// AddCommand adds a basic command to the registry. It returns false if the command
// is invalid or if the name was already registered.
func (c *CommandRegistry) AddCommand(cmd *BasicCommand) bool {
	if cmd == nil {
		return false
	}

	if _, found := c.cmdsMap[cmd.Name]; found {
		return false
	}

	if cmd.Name == "" {
		return false
	}

	if cmd.Handler == nil {
		return false
	}

	c.cmdsList = append(c.cmdsList, cmd)
	c.cmdsMap[cmd.Name] = cmd

	return true
}

// GetCommands returns a list of the registered BasicCommands.
func (c CommandRegistry) GetCommands() []*BasicCommand {
	return c.cmdsList
}

// GetCommand returns the BasicCommand registered with the given cmdName (or nil
// if there was no such cmd).
func (c CommandRegistry) GetCommand(cmdName string) *BasicCommand {
	return c.cmdsMap[cmdName]
}

// AddPattern adds a pattern command to the registry. It returns false if the
// pattern command is invalid or if the pattern name was already registered.
func (c *CommandRegistry) AddPattern(cmd *PatternCommand) bool {
	if cmd == nil {
		return false
	}

	if _, found := c.patternsMap[cmd.Name]; found {
		return false
	}

	if cmd.Name == "" {
		return false
	}

	if cmd.Handler == nil {
		return false
	}

	if cmd.Pattern == nil {
		return false
	}

	c.patternsList = append(c.patternsList, cmd)
	c.patternsMap[cmd.Name] = cmd

	return true
}

// GetPatterns returns a list of the registered PatternCommands.
func (c CommandRegistry) GetPatterns() []*PatternCommand {
	return c.patternsList
}

// GetPattern returns the Pattern Command registered with the given patternName
// (or nil if there was no such pattern).
func (c CommandRegistry) GetPattern(patternName string) *PatternCommand {
	return c.patternsMap[patternName]
}

// AddReactionHandler adds a reaction handler to the registry. It returns false
// if the reaction command is invalid or if the reaction command name was
// already registered.
func (c *CommandRegistry) AddReactionHandler(cmd *ReactionCommand) bool {
	if cmd == nil {
		return false
	}

	if cmd.Name == "" {
		return false
	}

	if _, found := c.reactionHandlerMap[cmd.Name]; found {
		return false
	}

	if cmd.Handler == nil {
		return false
	}

	c.reactionHandlerList = append(c.reactionHandlerList, cmd)
	c.reactionHandlerMap[cmd.Name] = cmd

	return true
}

// GetReactionHandlers returns a list of the registered ReactionCommands.
func (c CommandRegistry) GetReactionHandlers() []*ReactionCommand {
	return c.reactionHandlerList
}

// GetReactionHandler returns the Reaction Command registered with the given cmdName
// (or nil if there was no such handler).
func (c CommandRegistry) GetReactionHandler(cmdName string) *ReactionCommand {
	return c.reactionHandlerMap[cmdName]
}

// DefaultRegistry is the global registry instance used by default.
var DefaultRegistry = NewRegistry()

// AddCommand adds a command to the default registry.
func AddCommand(cmd *BasicCommand) bool {
	return DefaultRegistry.AddCommand(cmd)
}

// MustAddCommand adds a command to the default registry or panics.
func MustAddCommand(cmd *BasicCommand) {
	if added := DefaultRegistry.AddCommand(cmd); !added {
		panic(fmt.Sprintf("failed to add pattern: %v\n", cmd))
	}
}

// AddPattern adds a pattern command to the default registry.
func AddPattern(cmd *PatternCommand) bool {
	return DefaultRegistry.AddPattern(cmd)
}

// MustAddPattern adds a pattern command to the default registry or panics.
func MustAddPattern(cmd *PatternCommand) {
	if added := DefaultRegistry.AddPattern(cmd); !added {
		panic(fmt.Sprintf("failed to add pattern: %v\n", cmd))
	}
}

// AddReactionHandler adds a reaction command to the default registry.
func AddReactionHandler(cmd *ReactionCommand) bool {
	return DefaultRegistry.AddReactionHandler(cmd)
}

// AddReactionHandler adds a reaction command to the default registry or panics.
func MustAddReactionHandler(cmd *ReactionCommand) {
	if added := DefaultRegistry.AddReactionHandler(cmd); !added {
		panic(fmt.Sprintf("failed to add reaction handler: %v\n", cmd))
	}
}
