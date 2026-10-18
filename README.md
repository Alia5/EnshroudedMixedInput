# EnshroudedMixedInput

Mixed input for [Enshrouded](https://store.steampowered.com/app/1203220/Enshrouded/):
use a gamepad and mouse **at the same time**, without the game switching between them.

- Mouse input no longer switches the game to keyboard/mouse mode. Keyboard input still does.
- While in gamepad mode, mouse buttons, mouse look and the other keyboard/mouse bindings work alongside the gamepad.
- In gamepad mode, mouse look follows the gamepad "invert camera" setting.

## Installation

1. Download the latest release.
2. Extract it into the Enshrouded install directory, next to `enshrouded.exe`.
3. **Steam Deck / Linux (Proton) only:** set the game's launch options to

    ```text
    WINEDLLOVERRIDES="dinput8=n,b" %command%
    ```

To uninstall, delete `dinput8.dll` and `EnshroudedMixedInput.dll` from the game directory.

## Logs

The log is written to `EnshroudedMixedInput.log` in the game directory.

For debugging, set these environment variables (e.g. in the launch options):

- `EMI_LOG_LEVEL`: `trace`, `debug`, `info` (default), `warn` or `error`
- `EMI_LOG_CONSOLE=1`: opens a console window showing the log

## How it works

- `dinput8.dll` is a small proxy loader (`loader/`).
  It forwards DirectInput to the real system `dinput8.dll`
  and, inside `enshrouded.exe` only, loads `EnshroudedMixedInput.dll`.
  Set `EMI_SUPPRESS_LOADING` to skip loading it.
- `EnshroudedMixedInput.dll` is the mod, written in Go (`-buildmode=c-shared`).
  It finds game code by byte signatures and patches/detours it (`hook/`, `pattern/`, `mod/`).

## Building

Requirements (Windows): Go, [just](https://github.com/casey/just), CMake + MSVC (or LLVM-MinGW),
a cgo C compiler (e.g. LLVM-MinGW), and
[goversioninfo](https://github.com/josephspurrier/goversioninfo).

```sh
just build            # Debug build into dist/
just build Release    # Release build
just deploy           # Build and copy into the game directory (EMI_GAME_DIR)
just test             # Tests; signature tests run against enshrouded.exe when found
just lint
```

Signature tests look for `enshrouded.exe` in the default Steam library, or in `EMI_GAME_DIR`.

## License

[GPL-3.0](LICENSE.txt)

## LLMs

This project uses partial copies of my own (huma written) code from other projects...
However, parts of it are also LLM-slop. So, yeah, whatever ¯\\\_(ツ)_/¯
