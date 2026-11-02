package keen

// InputMode is keen::InputMode, reflected type at 0x141B69B80, values at 0x141B696B0.
type InputMode byte

const (
	InputModeKeyboardMouse InputMode = 0
	InputModeGamepad       InputMode = 1
)

// InputEventType is the input event type at event+0x0c, value names at 0x141FCB210.
type InputEventType byte

const (
	InputEventTypeMouseDown             InputEventType = 9
	InputEventTypeMouseUp               InputEventType = 10
	InputEventTypeMouseMove             InputEventType = 11
	InputEventTypeMouseRelativeMovement InputEventType = 12
	InputEventTypeMouseWheel            InputEventType = 13
	InputEventTypeMouseDoubleClick      InputEventType = 14
)

const (
	ActionsHeaderSize = 8
	ActionSize        = 280

	ActionKeyboardMousePrimaryBinding   = 32
	ActionKeyboardMouseSecondaryBinding = 84
)

const (
	BindingSourceKindOffset    = 0
	BindingSnapshotValueOffset = 32
)

type BindingSourceKind byte

const (
	BindingSourceUnbound               BindingSourceKind = 0
	BindingSourceKey                   BindingSourceKind = 1
	BindingSourceMouseButton           BindingSourceKind = 2
	BindingSourceMouseWheel            BindingSourceKind = 3
	BindingSourceMouseRelativeMovement BindingSourceKind = 4
	BindingSourceSpecial               BindingSourceKind = 5
	BindingSourceGamepadButton         BindingSourceKind = 6
	BindingSourceGamepadAxis           BindingSourceKind = 7
	BindingSourceGamepadStick          BindingSourceKind = 8
)

type GameplayAction byte

const (
	GameplayActionMovementLeft  GameplayAction = 0
	GameplayActionMovementRight GameplayAction = 1
	GameplayActionMovementUp    GameplayAction = 2
	GameplayActionMovementDown  GameplayAction = 3
	GameplayActionCameraLeft    GameplayAction = 4
	GameplayActionCameraRight   GameplayAction = 5
	GameplayActionCameraUp      GameplayAction = 6
	GameplayActionCameraDown    GameplayAction = 7

	GameplayActionCombatTargetSelect1dLeft  GameplayAction = 111
	GameplayActionCombatTargetSelect1dRight GameplayAction = 112
	GameplayActionCombatTargetSelect2dLeft  GameplayAction = 113
	GameplayActionCombatTargetSelect2dRight GameplayAction = 114
	GameplayActionCombatTargetSelect2dUp    GameplayAction = 115
	GameplayActionCombatTargetSelect2dDown  GameplayAction = 116
)
