package ai

import (
	"context"

	generator "github.com/mklimuk/vault-pilot/pkg/ai"
)

type GeneratorAdapter struct {
	generator.Generator
	name string
}

func NewGeneratorAdapter(gen generator.Generator, name string) Provider {
	return &GeneratorAdapter{
		Generator: gen,
		name:      name,
	}
}

func (a *GeneratorAdapter) Generate(ctx context.Context, req PromptRequest) (PromptResult, error) {
	output, err := a.Generator.GenerateText(ctx, req.Input)
	if err != nil {
		return PromptResult{}, err
	}
	return PromptResult{
		Output:    output,
		ModelUsed: a.name,
	}, nil
}

func (a *GeneratorAdapter) IsAvailable() bool {
	return a.Generator != nil
}

func (a *GeneratorAdapter) Name() string {
	return a.name
}
