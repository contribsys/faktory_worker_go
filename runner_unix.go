// +build linux freebsd netbsd openbsd dragonfly solaris illumos aix darwin

package faktory_worker

import (
	"os"
	"os/signal"
	"syscall"
)

var (
	SIGTERM os.Signal = syscall.SIGTERM
	SIGTSTP os.Signal = syscall.SIGTSTP
	SIGTTIN os.Signal = syscall.SIGTTIN
	SIGINT  os.Signal = os.Interrupt

	signalMap = map[os.Signal]string{
		SIGTERM: "terminate",
		SIGINT:  "terminate",
		SIGTSTP: "quiet",
		SIGTTIN: "dump",
	}
)

func hookSignals() chan os.Signal {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, SIGINT)
	signal.Notify(sigchan, SIGTERM)
	signal.Notify(sigchan, SIGTSTP)
	signal.Notify(sigchan, SIGTTIN)
	return sigchan
}
