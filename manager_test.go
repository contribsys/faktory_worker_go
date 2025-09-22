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

	// Create a temporary struct that will become a map after JSON serialization
	type tempStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	job := faktory.NewJob("test_job", tempStruct{Name: "John", Age: 30})

	err := mgr.InlineDispatch(job)
	assert.NoError(t, err)

	// Verify that the struct was converted to a map during serialization
	assert.Len(t, receivedArgs, 1)
	argMap, ok := receivedArgs[0].(map[string]interface{})
	assert.True(t, ok, "Expected argument to be converted to map[string]interface{}")
	assert.Equal(t, "John", argMap["name"])
	assert.Equal(t, float64(30), argMap["age"]) // JSON converts numbers to float64
}
