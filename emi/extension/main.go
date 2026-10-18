package main

import "C"

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/Alia5/EnshroudedMixedInput/console"
	"github.com/Alia5/EnshroudedMixedInput/hook"
	"github.com/Alia5/EnshroudedMixedInput/log"
	"github.com/Alia5/EnshroudedMixedInput/meta"
	"github.com/Alia5/EnshroudedMixedInput/mod"
)

const (
	logFileName = "EnshroudedMixedInput.log"

	logLevelEnv   = "EMI_LOG_LEVEL"
	logConsoleEnv = "EMI_LOG_CONSOLE"
)

func main() {}

func init() {
	go func() {
		hook.RecoverPanic(serve)
	}()
}

func serve() {
	library, libraryErr := modulePath()

	logFile := ""
	if libraryErr == nil {
		logFile = filepath.Join(filepath.Dir(library), logFileName)
	}

	var consoleErr error
	if os.Getenv(logConsoleEnv) != "" {
		consoleErr = console.Open()
	}

	logLevel := os.Getenv(logLevelEnv)

	_, _, err := log.SetupLogger(logLevel, logFile, log.ExtensionConsolePrefix)
	if err != nil {
		_, _, _ = log.SetupLogger(logLevel, "", log.ExtensionConsolePrefix)

		slog.Error("log file unavailable, logging to the console only", "file", logFile, "err", err)
	}

	if libraryErr != nil {
		slog.Error("own module path lookup failed", "err", libraryErr)
	}

	if consoleErr != nil {
		slog.Error("console unavailable", "err", consoleErr)
	}

	slog.Info("EnshroudedMixedInput loaded",
		"pid", os.Getpid(),
		"version", meta.Version,
		"commit", meta.Commit,
		"date", meta.Date,
		"library", library)

	installMods()
}

func installMods() {
	for _, m := range mod.All() {
		logger := slog.With("mod", m.Name())

		err := m.Install()
		if err != nil {
			logger.Error("mod install failed", "err", err)

			continue
		}

		logger.Info("mod installed")
	}
}
