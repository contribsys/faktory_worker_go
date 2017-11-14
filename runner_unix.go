// +build unix

package faktory_worker

import (
	"os"
	"os/signal"
	"syscall"
)

var (
	// SIGTERM is an alias for syscall.SIGTERM
	SIGTERM os.Signal = syscall.SIGTERM
	// SIGTSTP is an alias for syscall.SIGSTP
	SIGTSTP os.Signal = syscall.SIGTSTP
	// SIGINT is and alias for syscall.SIGINT
	SIGINT os.Signal = os.Interrupt
)

func hookSignals() chan os.Signal {
	sigchan := make(chan os.Signal)
	signal.Notify(sigchan, SIGINT)
	signal.Notify(sigchan, SIGTERM)
	signal.Notify(sigchan, SIGTSTP)
	return sigchan
}
