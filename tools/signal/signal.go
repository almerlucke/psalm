package signal

import (
	"os"
	"os/signal"
)

func WaitForSignal(sig ...os.Signal) os.Signal {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, sig...)
	return <-sigc
}
