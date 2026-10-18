package mixedinput

import (
	"errors"
	"fmt"

	"github.com/Alia5/EnshroudedMixedInput/game"
	"github.com/Alia5/EnshroudedMixedInput/log"
)

const name = "mixed_input"

var ErrNotImplemented = errors.New("not implemented yet")

// MixedInput keeps gamepad and mouse/keyboard input active at the same time,
// instead of the game switching between the two.
type MixedInput struct {
	log        *log.TraceLogger
	installErr error
}

func New() *MixedInput {
	return &MixedInput{
		log: log.TraceLoggerWith("mod", name),
	}
}

func (*MixedInput) Name() string {
	return name
}

func (m *MixedInput) Install() error {
	m.installErr = m.install()

	return m.installErr
}

func (m *MixedInput) Installed() (bool, error) {
	return false, m.installErr
}

func (m *MixedInput) Remove() error {
	return nil
}

func (m *MixedInput) install() error {
	module, image, err := game.Module()
	if err != nil {
		return err
	}

	m.log.Debug("game module mapped",
		"module", fmt.Sprintf("%#x", module),
		"size", fmt.Sprintf("%#x", len(image)))

	// TODO: locate the input device switching logic (see patterns.go) and hook it.
	return ErrNotImplemented
}
