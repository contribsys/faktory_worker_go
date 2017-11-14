// +build windows
package faktory_worker

import (
	"os"
	"os/signal"
	"syscall"
)

var (
	// SIGTERM is an alias for syscall.SIGTERM
	SIGTERM os.Signal = syscall.SIGTERM
	// SIGINT is and alias for syscall.SIGINT
	SIGINT  os.Signal = os.Interrupt
	SIGTSTP           = os.Signal(-1)
)

func hookSignals() chan os.Signal {
	sigchan := make(chan os.Signal)
	signal.Notify(sigchan, SIGINT)
	signal.Notify(sigchan, SIGTERM)
	return sigchan
}
