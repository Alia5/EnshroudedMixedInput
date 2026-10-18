#include <windows.h>

static HMODULE g_real_dinput8 = NULL;
static HMODULE g_extension = NULL;

static const char suppress_loading_env[] = "EMI_SUPPRESS_LOADING";

static void dir_relative_path(HMODULE self, const char *leaf, char *out)
{
    char base[MAX_PATH] = {0};
    GetModuleFileNameA(self, base, MAX_PATH);
    for (int i = (int)lstrlenA(base) - 1; i >= 0; --i)
    {
        if (base[i] == '\\' || base[i] == '/')
        {
            base[i] = '\0';
            break;
        }
    }
    wsprintfA(out, "%s\\%s", base, leaf);
}

static BOOL should_load_extension(void)
{
    if (GetEnvironmentVariableA(suppress_loading_env, NULL, 0) != 0)
    {
        return FALSE;
    }

    char path[MAX_PATH] = {0};
    GetModuleFileNameA(NULL, path, MAX_PATH);
    const char *leaf = path;
    for (const char *c = path; *c; ++c)
    {
        if (*c == '\\' || *c == '/')
        {
            leaf = c + 1;
        }
    }
    return lstrcmpiA(leaf, "enshrouded.exe") == 0;
}

static const char *const extension_file_names[] = {"EnshroudedMixedInput.dll", "emi.dll"};

static DWORD WINAPI load_extension(LPVOID param)
{
    char path[MAX_PATH];
    for (int i = 0; i < (int)ARRAYSIZE(extension_file_names) && !g_extension; ++i)
    {
        dir_relative_path((HMODULE)param, extension_file_names[i], path);
        g_extension = LoadLibraryA(path);
    }
    return 0;
}

#define PROXY(ret, fail, name, params, args)                    \
    ret WINAPI name params                                      \
    {                                                           \
        typedef ret(WINAPI * fn_t) params;                      \
        static fn_t real = NULL;                                \
        if (!real && g_real_dinput8)                            \
        {                                                       \
            real = (fn_t)GetProcAddress(g_real_dinput8, #name); \
        }                                                       \
        if (!real)                                              \
        {                                                       \
            return fail;                                        \
        }                                                       \
        return real args;                                       \
    }

PROXY(HRESULT, E_FAIL, DirectInput8Create, (HINSTANCE a, DWORD b, REFIID c, LPVOID *d, LPVOID e), (a, b, c, d, e))
PROXY(HRESULT, E_FAIL, DllCanUnloadNow, (void), ())
PROXY(HRESULT, E_FAIL, DllGetClassObject, (REFCLSID a, REFIID b, LPVOID *c), (a, b, c))
PROXY(HRESULT, E_FAIL, DllRegisterServer, (void), ())
PROXY(HRESULT, E_FAIL, DllUnregisterServer, (void), ())
PROXY(LPCVOID, NULL, GetdfDIJoystick, (void), ())

static void pin_self(void)
{
    HMODULE pinned = NULL;
    GetModuleHandleExA(GET_MODULE_HANDLE_EX_FLAG_PIN | GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS,
                       (LPCSTR)(void *)&load_extension, &pinned);
}

BOOL APIENTRY DllMain(HMODULE self, DWORD reason, LPVOID reserved)
{
    (void)reserved;
    if (reason == DLL_PROCESS_ATTACH)
    {
        DisableThreadLibraryCalls(self);
        pin_self();

        char path[MAX_PATH];
        GetSystemDirectoryA(path, MAX_PATH);
        lstrcatA(path, "\\dinput8.dll");
        g_real_dinput8 = LoadLibraryA(path);

        if (!should_load_extension())
        {
            return TRUE;
        }

        HANDLE t = CreateThread(NULL, 0, load_extension, self, 0, NULL);
        if (t)
        {
            CloseHandle(t);
        }
    }
    return TRUE;
}
