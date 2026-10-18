package sigtest

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Alia5/EnshroudedMixedInput/game"
	"github.com/Alia5/EnshroudedMixedInput/hook"
	"github.com/Alia5/EnshroudedMixedInput/pattern"
)

const DirEnv = "EMI_GAME_DIR"

const defaultDir = `C:\Program Files (x86)\Steam\steamapps\common\Enshrouded`

type Case struct {
	Name     string
	Pattern  pattern.Pattern
	TypeName string
	Err      error
}

func RunCases(t *testing.T, cases []Case) {
	t.Helper()

	RunModuleCases(t, game.ModuleName, cases)
}

func RunModuleCases(t *testing.T, name string, cases []Case) {
	t.Helper()

	module, image := LoadModule(t, name)

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			err := resolve(module, image, c)
			if !errors.Is(err, c.Err) {
				t.Fatalf("got %v, expected %v", err, c.Err)
			}
		})
	}
}

func resolve(module uintptr, image []byte, c Case) error {
	if c.TypeName == "" {
		_, err := hook.FindUniquePattern(module, image, c.Pattern)

		return err
	}

	table, err := hook.MethodTable(module, c.TypeName)
	if err != nil {
		return err
	}

	_, _, _, err = hook.FindUniqueSlot(module, image, table, c.Pattern.Index)

	return err
}

// LoadModule maps the named module from EMI_GAME_DIR (default: the Steam
// install) and skips the test when it is not there.
func LoadModule(t *testing.T, name string) (uintptr, []byte) {
	t.Helper()

	dir, set := os.LookupEnv(DirEnv)
	if !set {
		dir = defaultDir
	}

	path := filepath.Join(dir, name)

	_, err := os.Stat(path)
	if err != nil {
		if set {
			t.Fatalf("%s=%s: %v", DirEnv, dir, err)
		}

		t.Skipf("%s not found, set %s to run", path, DirEnv)
	}

	module, image, err := MapImage(path)
	if err != nil {
		t.Fatalf("map %s: %v", path, err)
	}

	return module, image
}
