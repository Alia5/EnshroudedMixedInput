package meta

import (
	"fmt"
	"runtime/debug"
	"time"
)

const (
	unknown     = "unknown"
	devVersion  = "dev"
	shortCommit = 7

	descriptionTemplate = `EnshroudedMixedInput (EMI)
  Version: %s (%s)
           %s
  `
)

var (
	Version = ""
	Commit  = ""
	Date    = ""
)

func Description() string {
	return fmt.Sprintf(descriptionTemplate, Version, Commit, Date)
}

func init() {
	info, ok := debug.ReadBuildInfo()
	if ok {
		fromBuildInfo(info)
	}

	if Version == "" {
		Version = devVersion
	}

	if Commit == "" {
		Commit = unknown
	}

	if Date == "" {
		Date = unknown
	}
}

func fromBuildInfo(info *debug.BuildInfo) {
	if Version == "" {
		Version = info.Main.Version

		if Version == "" || Version == "(devel)" {
			Version = devVersion
		}
	}

	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if Commit == "" {
				Commit = s.Value

				if len(Commit) > shortCommit {
					Commit = Commit[:shortCommit]
				}
			}

		case "vcs.time":
			if Date == "" {
				Date = s.Value

				t, err := time.Parse(time.RFC3339, s.Value)
				if err == nil {
					Date = t.Format("2006-01-02")
				}
			}
		}
	}
}
