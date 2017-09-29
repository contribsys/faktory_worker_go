package faktory_worker

import "testing"

func TestRegistration(t *testing.T) {
	mgr := NewManager()
	mgr.Register("somejob", func() Worker { return nil })
}
