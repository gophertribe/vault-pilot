package registry

import "context"

type MockPlugin struct {
	IDFunc       func() string
	RegisterFunc func(ctx context.Context, r Registrar) error
	StartFunc    func(ctx context.Context) error
	StopFunc     func(ctx context.Context) error
}

func (m *MockPlugin) ID() string {
	if m.IDFunc != nil {
		return m.IDFunc()
	}
	return "mock"
}

func (m *MockPlugin) Register(ctx context.Context, r Registrar) error {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(ctx, r)
	}
	return nil
}

func (m *MockPlugin) Start(ctx context.Context) error {
	if m.StartFunc != nil {
		return m.StartFunc(ctx)
	}
	return nil
}

func (m *MockPlugin) Stop(ctx context.Context) error {
	if m.StopFunc != nil {
		return m.StopFunc(ctx)
	}
	return nil
}
