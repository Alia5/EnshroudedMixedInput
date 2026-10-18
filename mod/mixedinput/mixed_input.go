package mixedinput

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"github.com/Alia5/EnshroudedMixedInput/game"
	"github.com/Alia5/EnshroudedMixedInput/hook"
	"github.com/Alia5/EnshroudedMixedInput/keen"
	"github.com/Alia5/EnshroudedMixedInput/log"
)

const name = "mixed_input"

var (
	ErrJumpTableBaseDisagrees                 = errors.New("updateInputModeFromEvent jump table is not relative to the image base")
	ErrMouseEventCaseDisagrees                = errors.New("updateInputModeFromEvent mouse event case does not lead to the KeyboardMouse store")
	ErrActiveStoreOutsideUpdateGameplayAction = errors.New("active state store is not in updateGameplayAction")
)

// MixedInput leaves the input mode unchanged on mouse input. In Gamepad mode the keyboard mouse
// bindings count alongside the gamepad ones.
type MixedInput struct {
	log        *log.TraceLogger
	installErr error

	mouseEventsKeepInputModePatch   hook.CodePatch
	updateGameplayActionDetour      hook.Detour
	addDirectionalActionInputDetour hook.Detour

	evaluateGameplayActionHalf uintptr
	actionEnabledOffset        uintptr
	actionActiveOffset         uintptr
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
	return m.mouseEventsKeepInputModePatch.Installed() &&
		m.updateGameplayActionDetour.Installed() &&
		m.addDirectionalActionInputDetour.Installed(), m.installErr
}

func (m *MixedInput) Remove() error {
	return errors.Join(
		m.addDirectionalActionInputDetour.Restore(),
		m.updateGameplayActionDetour.Restore(),
		m.mouseEventsKeepInputModePatch.Restore(),
	)
}

func (m *MixedInput) install() error {
	module, image, err := game.Module()
	if err != nil {
		return err
	}

	updateInputModeFromEvent, err := hook.FindUniquePattern(module, image, UpdateInputModeFromEventPattern)
	if err != nil {
		return fmt.Errorf("updateInputModeFromEvent: %w", err)
	}

	mouseEventEntries, noChangeEntries, err := mouseEventJumpTableEntries(module, image, updateInputModeFromEvent)
	if err != nil {
		return err
	}

	updateGameplayAction, err := hook.FindUniquePattern(module, image, UpdateGameplayActionPattern)
	if err != nil {
		return fmt.Errorf("updateGameplayAction: %w", err)
	}

	activeStore, err := hook.FindUniquePattern(module, image, UpdateGameplayActionActiveStorePattern)
	if err != nil {
		return fmt.Errorf("updateGameplayAction active state store: %w", err)
	}

	activeStoreFunction, err := hook.FunctionStart(module, image, activeStore)
	if err != nil {
		return fmt.Errorf("updateGameplayAction active state store: %w", err)
	}

	if activeStoreFunction != updateGameplayAction {
		return fmt.Errorf("%w: store at %#x, function at %#x", ErrActiveStoreOutsideUpdateGameplayAction, activeStore, updateGameplayAction)
	}

	addDirectionalActionInput, err := hook.FindUniquePattern(module, image, AddDirectionalActionInputPattern)
	if err != nil {
		return fmt.Errorf("addDirectionalActionInput: %w", err)
	}

	m.evaluateGameplayActionHalf = activeStore + evaluateCallEnd +
		uintptr(int32(binary.LittleEndian.Uint32(image[activeStore-module+evaluateCallRelOffset:])))
	m.actionEnabledOffset = uintptr(binary.LittleEndian.Uint32(image[updateGameplayAction-module+actionEnabledDispOffset:]))
	m.actionActiveOffset = uintptr(binary.LittleEndian.Uint32(image[activeStore-module+activeStoreDispOffset:]))

	err = m.mouseEventsKeepInputModePatch.Apply(mouseEventEntries, noChangeEntries)
	if err != nil {
		return fmt.Errorf("updateInputModeFromEvent jump table: %w", err)
	}

	err = m.updateGameplayActionDetour.Apply(updateGameplayAction, syscall.NewCallback(m.updateGameplayAction))
	if err != nil {
		return errors.Join(fmt.Errorf("updateGameplayAction: %w", err), m.mouseEventsKeepInputModePatch.Restore())
	}

	err = m.addDirectionalActionInputDetour.Apply(addDirectionalActionInput, syscall.NewCallback(m.addDirectionalActionInput))
	if err != nil {
		return errors.Join(fmt.Errorf("addDirectionalActionInput: %w", err),
			m.updateGameplayActionDetour.Restore(), m.mouseEventsKeepInputModePatch.Restore())
	}

	m.log.Debug("hooked input mode and gameplay action evaluation",
		"updateInputModeFromEvent", fmt.Sprintf("%#x", updateInputModeFromEvent),
		"mouseEventEntries", fmt.Sprintf("%#x", mouseEventEntries),
		"updateGameplayAction", fmt.Sprintf("%#x", updateGameplayAction),
		"evaluateGameplayActionHalf", fmt.Sprintf("%#x", m.evaluateGameplayActionHalf),
		"actionEnabledOffset", fmt.Sprintf("%#x", m.actionEnabledOffset),
		"actionActiveOffset", fmt.Sprintf("%#x", m.actionActiveOffset),
		"addDirectionalActionInput", fmt.Sprintf("%#x", addDirectionalActionInput))

	return nil
}

// mouseEventJumpTableEntries returns the address of the MouseDown..MouseDoubleClick jump table entries
// and their replacement, all pointing at the no change case.
func mouseEventJumpTableEntries(module uintptr, image []byte, updateInputModeFromEvent uintptr) (uintptr, []byte, error) {
	function := updateInputModeFromEvent - module

	imageBase := updateInputModeFromEvent + imageBaseLeaEnd +
		uintptr(int32(binary.LittleEndian.Uint32(image[function+imageBaseLeaDispOffset:])))
	if imageBase != module {
		return 0, nil, fmt.Errorf("%w: %#x, module at %#x", ErrJumpTableBaseDisagrees, imageBase, module)
	}

	noChangeCase := function + noChangeJumpEnd + uintptr(int8(image[function+noChangeJumpRelOffset]))
	keyboardMouseCase := function + keyboardMouseCaseOffset

	jumpTable := uintptr(binary.LittleEndian.Uint32(image[function+jumpTableDispOffset:]))
	first := jumpTable + uintptr(keen.InputEventTypeMouseDown-jumpTableFirstEventType)*jumpTableEntrySize
	count := int(keen.InputEventTypeMouseDoubleClick-keen.InputEventTypeMouseDown) + 1

	for i := range count {
		entry := uintptr(binary.LittleEndian.Uint32(image[first+uintptr(i)*jumpTableEntrySize:]))
		if entry != keyboardMouseCase {
			return 0, nil, fmt.Errorf("%w: event type %d leads to %#x, KeyboardMouse store at %#x",
				ErrMouseEventCaseDisagrees, keen.InputEventTypeMouseDown+keen.InputEventType(i), module+entry, module+keyboardMouseCase)
		}
	}

	return module + first, bytes.Repeat(binary.LittleEndian.AppendUint32(nil, uint32(noChangeCase)), count), nil
}

// In Gamepad mode the keyboard mouse half also counts towards the action's active state.
func (m *MixedInput) updateGameplayAction(action, mode, keyboardMouseModifiers, gamepadModifiers, index, time uintptr) uintptr {
	m.updateGameplayActionDetour.CallOriginal(action, mode, keyboardMouseModifiers, gamepadModifiers, index, time)

	if keen.InputMode(mode) != keen.InputModeGamepad || *(*byte)(unsafe.Pointer(action + m.actionEnabledOffset)) == 0 {
		return 0
	}

	keyboardMouseActive, _, _ := syscall.SyscallN(m.evaluateGameplayActionHalf, action, index, time)
	if byte(keyboardMouseActive) != 0 {
		*(*byte)(unsafe.Pointer(action + m.actionActiveOffset)) = 1
	}

	return 0
}

// In Gamepad mode the keyboard mouse binding adds its input after the gamepad one.
func (m *MixedInput) addDirectionalActionInput(input, actions, action, primary, dt uintptr) uintptr {
	result := m.addDirectionalActionInputDetour.CallOriginal(input, actions, action, primary, dt)

	mode := (*keen.InputMode)(unsafe.Pointer(actions))
	if *mode != keen.InputModeGamepad {
		return result
	}

	*mode = keen.InputModeKeyboardMouse
	m.addDirectionalActionInputDetour.CallOriginal(input, actions, action, primary, dt)
	*mode = keen.InputModeGamepad

	return result
}
