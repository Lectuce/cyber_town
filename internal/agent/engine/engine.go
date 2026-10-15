package engine

import (
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

type Config struct {
	Model            string
	MaxTokens        int64
	MaxRounds        int
	MaxModelRetries  int
	MaxContinuations int
	RetryBaseDelay   time.Duration
	TurnTimeout      time.Duration
}

type Engine struct {
	model           Model
	config          Config
	toolDefinitions map[string][]anthropic.ToolUnionParam
	toolHandlers    map[string]map[string]ToolHandler
}

func New(model Model,
	config Config,
	toolDefinitions map[string][]anthropic.ToolUnionParam,
	toolHandlers map[string]map[string]ToolHandler,
) (*Engine, error) {

	if model == nil {
		return nil, fmt.Errorf("model is required")
	}
	if config.Model == "" {
		return nil, fmt.Errorf("model name is required")
	}
	if config.MaxTokens <= 0 {
		return nil, fmt.Errorf("max tokens must be positive")
	}
	if config.MaxRounds <= 0 {
		return nil, fmt.Errorf("max rounds must be positive")
	}
	if config.TurnTimeout < 0 {
		return nil, fmt.Errorf("turn timeout cannot be negative")
	}

	err := validateTools(toolDefinitions, toolHandlers)
	if err != nil {
		return nil, err
	}

	return &Engine{
		model:           model,
		config:          config,
		toolDefinitions: toolDefinitions,
		toolHandlers:    toolHandlers,
	}, nil

}

func validateTools(
	definitions map[string][]anthropic.ToolUnionParam,
	handlers map[string]map[string]ToolHandler,
) error {

	for setName, toolDefinitions := range definitions {

		setHandlers, ok := handlers[setName]
		if !ok {
			return fmt.Errorf("tool handlers not found for set: %v", setName)
		}

		names := make(map[string]struct{}, 0)

		for _, definition := range toolDefinitions {

			if definition.OfTool == nil {
				continue
			}

			name := string(definition.OfTool.Name)

			if name == "" {
				return fmt.Errorf("tool name is empty in set: %v", setName)
			}

			_, ok := names[name]
			if ok {
				return fmt.Errorf("duplicate tool %v in set %v", name, setName)
			}

			names[name] = struct{}{}

			handler, ok := setHandlers[name]
			if !ok {
				return fmt.Errorf("handler not found for tool %v in set %v", name, setName)
			}

			if handler == nil {
				return fmt.Errorf("handler is nil for tool %v in set %v", name, setName)
			}

		}

		for name := range setHandlers {

			_, ok := names[name]
			if !ok {
				return fmt.Errorf("definition not found for handler %v in set %v", name, setName)
			}

		}

	}

	for setName := range handlers {
		_, ok := definitions[setName]
		if !ok {
			return fmt.Errorf("tool definitions not found for set: %s", setName)
		}
	}

	return nil
}
