package actors

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/mborders/logmatic"
)

// Logs to the terminal. Level options are: 0 fatal error (stack dump), 1 serious error (stack dump), 2 warning, 3 debug, 4 info, 5 trace (stack dump).
func LogCLI(message interface{}, level int) {
	l := logmatic.NewLogger()
	l.SetLevel(logmatic.TRACE)
	l.ExitOnFatal = true
	message = fmt.Sprint(message)
	switch level {
	case 5:
		debug.PrintStack()
		l.Trace("%v", message)
	case 4:
		l.Info("%v", message)
	case 3:
		l.Debug("%v", message)
	case 2:
		l.Warn("%v", message)
	case 1:
		debug.PrintStack()
		l.Error("%v", message)
	case 0:
		debug.PrintStack()
		l.Error("%v", message)
		Shutdown()
		go func() {
			LogCLI("Shutting down due to a consensus error. If any databases fail to close gracefully within 120 seconds they will be destroyed.", 4)
			//If everything goes well, closing the interrupt channel should shutdown cleanly before terminating.
			//If something goes wrong we kill the process
			time.Sleep(time.Second * 120)
			println("Something didn't shutdown cleanly. In addition to whatever problem caused this, our " +
				"data is probably corrupt like our leaders.")
			os.Exit(0)
		}()
	}

}
