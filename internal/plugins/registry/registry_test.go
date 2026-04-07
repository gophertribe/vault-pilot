package registry

import (
	"context"
	"testing"

	"github.com/mklimuk/vault-pilot/internal/core/commands"
	"github.com/mklimuk/vault-pilot/internal/core/events"
	"github.com/mklimuk/vault-pilot/internal/core/scheduler"
)

func TestPluginInterface(t *testing.T) {
	testPlugin := &MockPlugin{
		IDFunc: func() string { return "test-plugin" },
		RegisterFunc: func(ctx context.Context, r Registrar) error {
			r.OnEvent(events.TypeMailReceived, func(ctx context.Context, evt events.Event) error {
				return nil
			})
			r.OnCommand("test", func(ctx context.Context, cmd commands.Command) error {
				return nil
			})
			r.AddSchedule(scheduler.ScheduleSpec{
				ID:       "test-schedule",
				PluginID: "test-plugin",
				Kind:     scheduler.KindInterval,
				Expr:     "1h",
			}, func(ctx context.Context, spec scheduler.ScheduleSpec) error {
				return nil
			})
			return nil
		},
	}

	var _ Plugin = testPlugin
}

func TestRegistry_RegisterPlugin(t *testing.T) {
	r := NewRegistry(nil)

	p1 := &MockPlugin{IDFunc: func() string { return "plugin1" }}
	p2 := &MockPlugin{IDFunc: func() string { return "plugin2" }}

	if err := r.RegisterPlugin(p1); err != nil {
		t.Fatalf("register plugin1: %v", err)
	}
	if err := r.RegisterPlugin(p2); err != nil {
		t.Fatalf("register plugin2: %v", err)
	}
	if err := r.RegisterPlugin(p1); err == nil {
		t.Fatal("expected error for duplicate plugin registration")
	}
}

func TestRegistry_EventSubscriptions(t *testing.T) {
	r := NewRegistry(nil).(*pluginRegistry)

	handler := func(ctx context.Context, evt events.Event) error { return nil }
	r.OnEvent(events.TypeMailReceived, handler)
	r.OnEvent(events.TypeGTDTaskCreated, handler)

	subs := r.GetEventSubscriptions()
	if len(subs) != 2 {
		t.Fatalf("expected 2 subscriptions, got %d", len(subs))
	}
}

func TestRegistry_CommandRegistrations(t *testing.T) {
	r := NewRegistry(nil).(*pluginRegistry)

	handler := func(ctx context.Context, cmd commands.Command) error { return nil }
	r.OnCommand("inbox", handler)
	r.OnCommand("software", handler)

	regs := r.GetCommandRegistrations()
	if len(regs) != 2 {
		t.Fatalf("expected 2 registrations, got %d", len(regs))
	}
}
