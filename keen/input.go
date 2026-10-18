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
