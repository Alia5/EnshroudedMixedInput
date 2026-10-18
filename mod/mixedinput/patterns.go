package mixedinput

import "github.com/Alia5/EnshroudedMixedInput/pattern"

const (
	noChangeJumpRelOffset   = 13 // 77 rel8
	noChangeJumpEnd         = 14
	imageBaseLeaDispOffset  = 17 // 4C 8D 15 rel32
	imageBaseLeaEnd         = 21
	jumpTableDispOffset     = 27 // 45 8B 84 82 disp32
	keyboardMouseCaseOffset = 37 // 80 39 00
	jumpTableFirstEventType = 1  // lea eax, [r9-1]
	jumpTableEntrySize      = 4

	actionEnabledDispOffset = 13 // 80 B9 disp32 00
	evaluateCallRelOffset   = 32 // E8 rel32
	evaluateCallEnd         = 36
	activeStoreDispOffset   = 38 // 88 86 disp32
)

// void updateInputModeFromEvent(actions, const event *)
// actions[0] is the keen::InputMode, event+0x0c the input event type.
//
//	44 0F B6 4A 0C        movzx r9d, byte ptr [rdx+0Ch]
//	41 8D 41 FF           lea   eax, [r9-1]
//	83 F8 0F              cmp   eax, 0Fh
//	77 rel8               ja    no change
//	4C 8D 15 rel32        lea   r10, image base
//	48 98                 cdqe
//	45 8B 84 82 disp32    mov   r8d, [r10+rax*4+jump table]
//	4D 03 C2              add   r8, r10
//	41 FF E0              jmp   r8
//	80 39 00              cmp   byte ptr [rcx], KeyboardMouse   ; KeyDown..MouseDoubleClick
//	74 rel8               jz    no change
//	FE 81 disp32          inc   byte ptr [rcx+mode change count]
//	C6 01 00              mov   byte ptr [rcx], KeyboardMouse
//	C3                    ret

var UpdateInputModeFromEventPattern = pattern.MustParse(
	"44 0F B6 4A 0C 41 8D 41 FF 83 F8 0F 77 ?? 4C 8D 15 ?? ?? ?? ?? 48 98 45 8B 84 82 ?? ?? ?? ?? " +
		"4D 03 C2 41 FF E0 80 39 00 74 ?? FE 81 ?? ?? ?? ?? C6 01 00 C3")

// void updateGameplayAction(action, InputMode mode, uint32 keyboardMouseModifiers, uint32 gamepadModifiers,
// uint8 index, int64 time)
//
//	88 54 24 10           mov   [rsp+10h], dl
//	53 56 57              push  rbx, rsi, rdi
//	48 83 EC 30           sub   rsp, 30h
//	80 B9 disp32 00       cmp   byte ptr [rcx+enabled], 0
//	41 8B F9              mov   edi, r9d
//	41 8B D8              mov   ebx, r8d
//	48 8B F1              mov   rsi, rcx
//	0F 84                 jz    return

var UpdateGameplayActionPattern = pattern.MustParse(
	"88 54 24 10 53 56 57 48 83 EC 30 80 B9 ?? ?? ?? ?? 00 41 8B F9 41 8B D8 48 8B F1 0F 84")

// updateGameplayAction tail: active state from the half the mode selects.
// rsi = action, the keyboard mouse half at action+0; r12 = gamepad half; rbx = time.
//
//	88 86 disp32          mov   [rsi+previously active], al
//	85 D2                 test  edx, edx                  ; mode
//	74 rel8               jz    keyboard mouse half
//	83 FA 01              cmp   edx, Gamepad
//	75 rel8               jnz   skip
//	49 8B CC              mov   rcx, r12
//	EB rel8               jmp   evaluate
//	48 8B CE              mov   rcx, rsi
//	0F B6 54 24 disp8     movzx edx, index
//	4C 8B C3              mov   r8, rbx
//	E8 rel32              call  evaluateGameplayActionHalf(half, index, time)
//	88 86 disp32          mov   [rsi+active], al

var UpdateGameplayActionActiveStorePattern = pattern.MustParse(
	"88 86 ?? ?? ?? ?? 85 D2 74 ?? 83 FA 01 75 ?? 49 8B CC EB ?? 48 8B CE 0F B6 54 24 ?? 4C 8B C3 " +
		"E8 ?? ?? ?? ?? 88 86 ?? ?? ?? ??")

// void addDirectionalActionInput(float *input, actions, uint8 action, bool primary, int64 dt)
// Callers: computeCameraActionInput (CameraLeft..CameraDown), computeMovementInput (MovementLeft..MovementDown).
//
//	48 89 5C 24 08        mov   [rsp+8], rbx
//	48 89 6C 24 10        mov   [rsp+10h], rbp
//	48 89 74 24 18        mov   [rsp+18h], rsi
//	57                    push  rdi
//	48 83 EC 20           sub   rsp, 20h
//	48 8B DA              mov   rbx, rdx
//	41 0F B6 F0           movzx esi, r8b
//	48 8B F9              mov   rdi, rcx
//	40 0F B6 D6           movzx edx, sil
//	48 8B CB              mov   rcx, rbx
//	41 0F B6 E9           movzx ebp, r9b
//	E8 rel32              call  isGameplayActionActive(actions, action)
//	84 C0                 test  al, al
//	0F 84 rel32           jz    return
//	44 0F B6 C5           movzx r8d, bpl
//	40 0F B6 D6           movzx edx, sil
//	48 8B CB              mov   rcx, rbx
//	E8 rel32              call  getGameplayActionBinding(actions, action, primary)   ; InputMode half
//	4C 8B 4C 24 disp8     mov   r9, dt
//	45 33 C0              xor   r8d, r8d
//	40 0F B6 D6           movzx edx, sil
//	48 8B C8              mov   rcx, rax
//	48 8B D8              mov   rbx, rax
//	E8                    call  getBindingAxisValue

var AddDirectionalActionInputPattern = pattern.MustParse(
	"48 89 5C 24 08 48 89 6C 24 10 48 89 74 24 18 57 48 83 EC 20 48 8B DA 41 0F B6 F0 48 8B F9 " +
		"40 0F B6 D6 48 8B CB 41 0F B6 E9 E8 ?? ?? ?? ?? 84 C0 0F 84 ?? ?? ?? ?? 44 0F B6 C5 40 0F B6 D6 " +
		"48 8B CB E8 ?? ?? ?? ?? 4C 8B 4C 24 ?? 45 33 C0 40 0F B6 D6 48 8B C8 48 8B D8 E8")
