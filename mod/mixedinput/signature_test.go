package mixedinput_test

import (
	"testing"

	"github.com/Alia5/EnshroudedMixedInput/hook"
	"github.com/Alia5/EnshroudedMixedInput/pattern"
	"github.com/Alia5/EnshroudedMixedInput/testing/sigtest"
)

func TestSignatures(t *testing.T) {
	sigtest.RunCases(t, []sigtest.Case{
		{
			Name:    "FAIL_AmbiguousPattern",
			Pattern: pattern.MustParse("48 8B 07"),
			Err:     hook.ErrAmbiguousMatch,
		},
		{
			Name:    "FAIL_UnknownFunctionPattern",
			Pattern: pattern.MustParse("de ad be ef de ad be ef de ad be ef de ad be ef"),
			Err:     hook.ErrPatternNotFound,
		},
	})
}
