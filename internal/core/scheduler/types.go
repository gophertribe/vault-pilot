package scheduler

import (
	"context"
)

type ScheduleKind string

const (
	KindCron     ScheduleKind = "cron"
	KindInterval ScheduleKind = "interval"
)

type ScheduleSpec struct {
	ID       string       `json:"id"`
	PluginID string       `json:"plugin_id"`
	Kind     ScheduleKind `json:"kind"`
	Expr     string       `json:"expr"`
	Timezone string       `json:"timezone"`
	Enabled  bool         `json:"enabled"`
}

type ScheduledCallback func(ctx context.Context, spec ScheduleSpec) error

type Service interface {
	Add(spec ScheduleSpec, callback ScheduledCallback) error
	Remove(specID string) error
	Start() error
	Stop() error
}
