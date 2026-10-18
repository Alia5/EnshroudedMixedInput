package game

import (
	"github.com/Alia5/EnshroudedMixedInput/hook"
)

const ModuleName = "enshrouded.exe"

// Module returns the base address and mapped image of the game executable.
func Module() (uintptr, []byte, error) {
	module, err := hook.Module(ModuleName)
	if err != nil {
		return 0, nil, err
	}

	image, err := hook.Image(module)
	if err != nil {
		return 0, nil, err
	}

	return module, image, nil
}
