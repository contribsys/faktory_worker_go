package faktory_worker

import (
	"context"
	"errors"
	"log"
	"os"
	"syscall"
	"testing"
	"time"

	faktory "github.com/contribsys/faktory/client"
	"github.com/contribsys/faktory/util"
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
	err = mgr.setUpWorkerProcess()
	assert.NoError(t, err)

	startupCalled := false
	mgr.On(Startup, func(m *Manager) error {
		startupCalled = true
		assert.NotNil(t, m)
		return nil
	})
	mgr.fireEvent(Startup)
	assert.True(t, startupCalled)

	withServer(t, "oss", mgr, func(cl *faktory.Client) error {
		info, err := cl.Info()
		assert.NoError(t, err)
		sz := info["faktory"].(map[string]interface{})["tasks"].(map[string]interface{})["Workers"].(map[string]interface{})["size"].(float64)
		assert.EqualValues(t, startsz+1, sz)

		return nil
	})

	assert.Equal(t, "", mgr.handleEvent("quiet"))
	time.Sleep(1 * time.Millisecond)
	assert.Equal(t, "quiet", mgr.handleEvent("quiet"))

	devnull, err := os.OpenFile("/dev/null", os.O_WRONLY, 0)
	assert.NoError(t, err)

	logg := &StdLogger{
		log.New(devnull, "", 0),
	}

	mgr.Logger = logg
	assert.Equal(t, "", mgr.handleEvent("dump"))

	terminateCalled := false
	mgr.On(Shutdown, func(m *Manager) error {
		terminateCalled = true
		assert.NotNil(t, m)
		return nil
	})
	mgr.Terminate(false)
	assert.Equal(t, true, terminateCalled)
	// calling terminate again should be a noop
	terminateCalled = false
	mgr.Terminate(false)
	assert.Equal(t, false, terminateCalled)
}

func withServer(t *testing.T, lvl string, mgr *Manager, fn func(cl *faktory.Client) error) {
	err := mgr.with(func(cl *faktory.Client) error {
		if lvl == "oss" {
			return fn(cl)
		}

		hash, err := cl.Info()
		if err != nil {
			return err
		}
		desc := hash["server"].(map[string]interface{})["description"].(string)
		if lvl == "ent" && desc == "Faktory Enterprise" {
			return fn(cl)
		} else if lvl == "pro" && desc != "Faktory" {
			return fn(cl)
		}
		return nil
	})

	if errors.Is(err, syscall.ECONNREFUSED) {
		util.Debug("Server not running, skipping...")
		return
	} else {
		assert.NoError(t, err)
	}
}

func TestInlineDispatchArgsSerialization(t *testing.T) {
	mgr := NewManager()

	var receivedArgs []interface{}
	mgr.Register("test_job", func(ctx context.Context, args ...interface{}) error {
		receivedArgs = args
		return nil
	})

	// Test with various argument types that change during JSON serialization
	job := faktory.NewJob("test_job",
		int(42),                                // becomes float64
		int64(123),                             // becomes float64
		float32(3.14),                          // becomes float64
		"hello",                                // remains string
		true,                                   // remains bool
		map[string]interface{}{"key": "value"}, // remains map
	)

	err := mgr.InlineDispatch(job)
	assert.NoError(t, err)

	// Verify that numeric types were converted to float64 (JSON default)
	assert.Equal(t, float64(42), receivedArgs[0])
	assert.Equal(t, float64(123), receivedArgs[1])
	assert.Equal(t, float64(3.14), receivedArgs[2])
	assert.Equal(t, "hello", receivedArgs[3])
	assert.Equal(t, true, receivedArgs[4])
	assert.Equal(t, map[string]interface{}{"key": "value"}, receivedArgs[5])
}

func TestInlineDispatchNonRegisteredJob(t *testing.T) {
	mgr := NewManager()

	job := faktory.NewJob("non_existent_job", "arg1", "arg2")

	err := mgr.InlineDispatch(job)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job not registered")
}

func TestSerializeArgs(t *testing.T) {
	// Test the helper function directly
	originalArgs := []interface{}{
		int(42),
		int64(123),
		float32(3.14),
		"hello",
		true,
		map[string]interface{}{"key": "value"},
	}

	serializedArgs, err := serializeArgs(originalArgs)
	assert.NoError(t, err)

	// Verify that numeric types were converted to float64
	assert.Equal(t, float64(42), serializedArgs[0])
	assert.Equal(t, float64(123), serializedArgs[1])
	assert.Equal(t, float64(3.14), serializedArgs[2])
	assert.Equal(t, "hello", serializedArgs[3])
	assert.Equal(t, true, serializedArgs[4])
	assert.Equal(t, map[string]interface{}{"key": "value"}, serializedArgs[5])
}
