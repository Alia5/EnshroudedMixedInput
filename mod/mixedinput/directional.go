package mixedinput

import (
	"math"

	"github.com/Alia5/EnshroudedMixedInput/keen"
)

type DirectionalInput [8]float32

const (
	directionalSlotButtons       = 0
	directionalSlotGamepadAnalog = 2
	directionalSlotMouse         = 4
	directionalSlotSpecial       = 6
	directionalSlotNone          = -1
)

func directionalSlot(kind keen.BindingSourceKind) int {
	switch kind {
	case keen.BindingSourceKey, keen.BindingSourceMouseButton, keen.BindingSourceMouseWheel, keen.BindingSourceGamepadButton:
		return directionalSlotButtons
	case keen.BindingSourceGamepadAxis, keen.BindingSourceGamepadStick:
		return directionalSlotGamepadAnalog
	case keen.BindingSourceMouseRelativeMovement:
		return directionalSlotMouse
	case keen.BindingSourceSpecial:
		return directionalSlotSpecial
	default:
		return directionalSlotNone
	}
}

const (
	cameraAxisScale          = math.Pi / 2
	cameraMouseDivisor       = 800
	targetSelectMouseScale   = 0.2
	targetSelectMouseDivisor = 5000
	nanosecondsPerSecond     = 1e9
)

// bindingAxisValue mirrors getBindingAxisValue: snapshot value, scaled per action group.
func bindingAxisValue(action keen.GameplayAction, kind keen.BindingSourceKind, snapshot float32, dt int64) float32 {
	seconds := float32(float64(dt) / nanosecondsPerSecond)

	switch action {
	case keen.GameplayActionCameraLeft, keen.GameplayActionCameraRight, keen.GameplayActionCameraUp, keen.GameplayActionCameraDown,
		keen.GameplayActionCombatTargetSelect1dLeft, keen.GameplayActionCombatTargetSelect1dRight:
		switch kind {
		case keen.BindingSourceUnbound:
			return 0
		case keen.BindingSourceMouseRelativeMovement:
			return snapshot * cameraAxisScale / cameraMouseDivisor
		default:
			return snapshot * cameraAxisScale * seconds
		}
	case keen.GameplayActionCombatTargetSelect2dLeft, keen.GameplayActionCombatTargetSelect2dRight,
		keen.GameplayActionCombatTargetSelect2dUp, keen.GameplayActionCombatTargetSelect2dDown:
		if kind == keen.BindingSourceMouseRelativeMovement {
			return targetSelectMouseScale * snapshot / seconds / targetSelectMouseDivisor
		}

		return snapshot
	default:
		return snapshot
	}
}

func addDirectional(input *DirectionalInput, slot int, action keen.GameplayAction, value float32) {
	if slot < 0 {
		return
	}

	switch action {
	case keen.GameplayActionMovementLeft, keen.GameplayActionCameraRight,
		keen.GameplayActionCombatTargetSelect1dRight, keen.GameplayActionCombatTargetSelect2dRight:
		input[slot] -= value
	case keen.GameplayActionMovementRight, keen.GameplayActionCameraLeft,
		keen.GameplayActionCombatTargetSelect1dLeft, keen.GameplayActionCombatTargetSelect2dLeft:
		input[slot] += value
	case keen.GameplayActionMovementUp, keen.GameplayActionCameraUp, keen.GameplayActionCombatTargetSelect2dUp:
		input[slot+1] += value
	case keen.GameplayActionMovementDown, keen.GameplayActionCameraDown, keen.GameplayActionCombatTargetSelect2dDown:
		input[slot+1] -= value
	}
}
