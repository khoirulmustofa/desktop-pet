package app

import (
	"os"

	"desktop-pet/internal/window"
)

// syscallStderr mirrors stderr for -H=windowsgui builds (harmless when null).
var stderr = os.Stderr

// cursorPos returns the cursor screen position.
func cursorPos() (int, int) {
	return window.GetCursorPos()
}

// executablePath returns the current executable path.
func executablePath() (string, error) {
	return os.Executable()
}
