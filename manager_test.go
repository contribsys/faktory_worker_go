package faktory_worker

import (
	"testing"

	faktory "github.com/contribsys/faktory/client"
	"github.com/stretchr/testify/assert"
)

func TestManagerNoServer(t *testing.T) {
	t.Parallel()

	mgr := NewManager()
	mgr.setUpWorkerProcess()

	startupCalled := false
	mgr.On(Startup, func(m *Manager) error {
		startupCalled = true
		assert.NotNil(t, m)
		return nil
	})
	mgr.fireEvent(Startup)
	assert.True(t, startupCalled)

	called := false
	mgr.with(func(cl *faktory.Client) error {
		called = true
		assert.NotNil(t, cl)
		return nil
	})
	assert.True(t, called)

	mgr.Quiet()
	mgr.Terminate(false)
}
