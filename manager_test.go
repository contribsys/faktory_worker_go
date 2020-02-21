package faktory_worker

import (
	"errors"
	"fmt"
	"syscall"
	"testing"

	faktory "github.com/contribsys/faktory/client"
	"github.com/stretchr/testify/assert"
)

func TestManagerSetup(t *testing.T) {
	clx, err := faktory.Open()
	startsz := 0.0
	if err == nil {
		info, err := clx.Info()
		if err == nil {
			startsz = info["faktory"].(map[string]interface{})["tasks"].(map[string]interface{})["Workers"].(map[string]interface{})["size"].(float64)
		}
	}

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

	withServer(t, mgr, func(cl *faktory.Client) error {
		info, err := cl.Info()
		assert.NoError(t, err)
		sz := info["faktory"].(map[string]interface{})["tasks"].(map[string]interface{})["Workers"].(map[string]interface{})["size"].(float64)
		assert.EqualValues(t, startsz+1, sz)

		return nil
	})

	mgr.Quiet()
	mgr.Terminate(false)
}

func withServer(t *testing.T, mgr *Manager, fn func(cl *faktory.Client) error) {
	err := mgr.with(func(cl *faktory.Client) error {
		return fn(cl)
	})

	if errors.Is(err, syscall.ECONNREFUSED) {
		fmt.Println("Server not running, skipping...")
		return
	} else {
		assert.NoError(t, err)
	}
}
