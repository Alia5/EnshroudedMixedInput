package console

import (
	"os"

	"golang.org/x/sys/windows"
)

const (
	conout = "CONOUT$"

	enableVirtualTerminalProcessing = 0x0004
)

var (
	kernel32     = windows.NewLazySystemDLL("kernel32.dll")
	allocConsole = kernel32.NewProc("AllocConsole")
)

// Open attaches a new console window to the game process and points
// os.Stdout and os.Stderr at it, so the console log handlers show up.
func Open() error {
	ok, _, err := allocConsole.Call()
	if ok == 0 {
		return err
	}

	out, err := os.OpenFile(conout, os.O_WRONLY, 0)
	if err != nil {
		return err
	}

	var mode uint32

	handle := windows.Handle(out.Fd())
	if windows.GetConsoleMode(handle, &mode) == nil {
		_ = windows.SetConsoleMode(handle, mode|enableVirtualTerminalProcessing)
	}

	os.Stdout = out
	os.Stderr = out

	return nil
}
