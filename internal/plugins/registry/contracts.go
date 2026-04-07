package registry

import (
	"context"

	"github.com/mklimuk/vault-pilot/internal/core/commands"
	"github.com/mklimuk/vault-pilot/internal/core/events"
	"github.com/mklimuk/vault-pilot/internal/core/scheduler"
)

type Plugin interface {
	ID() string
	Register(ctx context.Context, r Registrar) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type Registrar interface {
	OnEvent(eventType string, handler events.EventHandler)
	OnCommand(name string, handler commands.CommandHandler)
	AddSchedule(spec scheduler.ScheduleSpec, callback scheduler.ScheduledCallback) error
}

type Registry interface {
	Registrar
	RegisterPlugin(p Plugin) error
	StartAll(ctx context.Context) error
	StopAll(ctx context.Context) error
}
