package mixedinput_test

import (
	"testing"

	"github.com/Alia5/EnshroudedMixedInput/hook"
	"github.com/Alia5/EnshroudedMixedInput/mod/mixedinput"
	"github.com/Alia5/EnshroudedMixedInput/pattern"
	"github.com/Alia5/EnshroudedMixedInput/testing/sigtest"
)

func TestSignatures(t *testing.T) {
	sigtest.RunCases(t, []sigtest.Case{
		{
			Name:    "SUCCESS_UpdateInputModeFromEvent",
			Pattern: mixedinput.UpdateInputModeFromEventPattern,
			Err:     nil,
		},
		{
			Name:    "SUCCESS_UpdateGameplayAction",
			Pattern: mixedinput.UpdateGameplayActionPattern,
			Err:     nil,
		},
		{
			Name:    "SUCCESS_UpdateGameplayActionActiveStore",
			Pattern: mixedinput.UpdateGameplayActionActiveStorePattern,
			Err:     nil,
		},
		{
			Name:    "SUCCESS_AddDirectionalActionInput",
			Pattern: mixedinput.AddDirectionalActionInputPattern,
			Err:     nil,
		},
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
