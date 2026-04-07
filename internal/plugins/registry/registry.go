package registry

import (
	"context"
	"fmt"
	"sync"

	"github.com/mklimuk/vault-pilot/internal/core/commands"
	"github.com/mklimuk/vault-pilot/internal/core/events"
	"github.com/mklimuk/vault-pilot/internal/core/scheduler"
)

type eventSubscription struct {
	eventType string
	handler   events.EventHandler
}

type commandRegistration struct {
	name    string
	handler commands.CommandHandler
}

type scheduleRegistration struct {
	spec     scheduler.ScheduleSpec
	callback scheduler.ScheduledCallback
}

type pluginRegistry struct {
	mu sync.RWMutex

	plugins          map[string]Plugin
	eventSubs        []eventSubscription
	commandRegs      []commandRegistration
	scheduleRegs     []scheduleRegistration
	schedulerService scheduler.Service
}

func NewRegistry(sched scheduler.Service) Registry {
	return &pluginRegistry{
		plugins:          make(map[string]Plugin),
		schedulerService: sched,
	}
}

func (r *pluginRegistry) OnEvent(eventType string, handler events.EventHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.eventSubs = append(r.eventSubs, eventSubscription{
		eventType: eventType,
		handler:   handler,
	})
}

func (r *pluginRegistry) OnCommand(name string, handler commands.CommandHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commandRegs = append(r.commandRegs, commandRegistration{
		name:    name,
		handler: handler,
	})
}

func (r *pluginRegistry) AddSchedule(spec scheduler.ScheduleSpec, callback scheduler.ScheduledCallback) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.scheduleRegs = append(r.scheduleRegs, scheduleRegistration{
		spec:     spec,
		callback: callback,
	})
	return nil
}

func (r *pluginRegistry) RegisterPlugin(p Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[p.ID()]; exists {
		return fmt.Errorf("plugin %s already registered", p.ID())
	}
	r.plugins[p.ID()] = p
	return nil
}

func (r *pluginRegistry) StartAll(ctx context.Context) error {
	r.mu.RLock()
	plugins := make([]Plugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		plugins = append(plugins, p)
	}
	r.mu.RUnlock()

	for _, p := range plugins {
		if err := p.Start(ctx); err != nil {
			return fmt.Errorf("start plugin %s: %w", p.ID(), err)
		}
	}
	return nil
}

func (r *pluginRegistry) StopAll(ctx context.Context) error {
	r.mu.RLock()
	plugins := make([]Plugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		plugins = append(plugins, p)
	}
	r.mu.RUnlock()

	var lastErr error
	for _, p := range plugins {
		if err := p.Stop(ctx); err != nil {
			lastErr = fmt.Errorf("stop plugin %s: %w", p.ID(), err)
		}
	}
	return lastErr
}

func (r *pluginRegistry) GetEventSubscriptions() []eventSubscription {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]eventSubscription, len(r.eventSubs))
	copy(result, r.eventSubs)
	return result
}

func (r *pluginRegistry) GetCommandRegistrations() []commandRegistration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]commandRegistration, len(r.commandRegs))
	copy(result, r.commandRegs)
	return result
}

func (r *pluginRegistry) GetScheduleRegistrations() []scheduleRegistration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]scheduleRegistration, len(r.scheduleRegs))
	copy(result, r.scheduleRegs)
	return result
}
