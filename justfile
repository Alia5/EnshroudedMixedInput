set windows-shell := ["powershell.exe", "-NoProfile", "-Command"]

binary_name := "EnshroudedMixedInput"
extension_pkg := "./emi/extension"
loader_dir := "loader"
dist_dir := "./dist/"
mkdir_p := if os_family() == "windows" { "New-Item -ItemType Directory -Force" } else { "mkdir -p" }
rm_rf := if os_family() == "windows" { "Remove-Item -Recurse -Force -ErrorAction 0" } else { "rm -rf" }
game_dir := env_var_or_default("EMI_GAME_DIR", "C:\\Program Files (x86)\\Steam\\steamapps\\common\\Enshrouded")
git_version := if os_family() == "windows" { `git describe --tags --always 2>$null; if ($LASTEXITCODE) { 'dev' }` } else { `git describe --tags --always 2>/dev/null || echo dev` }
raw_version := env_var_or_default("VERSION", git_version)
version := if raw_version =~ '^v?\d+\.\d+\.\d+' { raw_version } else { "v0.0.0-" + raw_version }
version_major := replace_regex(version, '^v?(\d+)\..*$', '$1')
version_minor := replace_regex(version, '^v?\d+\.(\d+)\..*$', '$1')
version_patch := replace_regex(version, '^v?\d+\.\d+\.(\d+).*$', '$1')
commits_since_tag := replace_regex(version, '^.*?(?:-(\d+)-g[0-9a-f]+)?$', '${1}')
version_build := if commits_since_tag == "" { "0" } else { commits_since_tag }
fixed_version := version_major + "," + version_minor + "," + version_patch + "," + version_build
versioninfo_flags := "-64 -ver-major " + version_major + " -ver-minor " + version_minor + " -ver-patch " + version_patch + " -ver-build " + version_build + " -product-ver-major " + version_major + " -product-ver-minor " + version_minor + " -product-ver-patch " + version_patch + " -product-ver-build " + version_build + " -file-version " + trim_start_match(version, "v") + " -product-version " + trim_start_match(version, "v")
commit := if os_family() == "windows" { `git rev-parse --short HEAD 2>$null; if ($LASTEXITCODE) { 'unknown' }` } else { `git rev-parse --short HEAD 2>/dev/null || echo unknown` }
build_time := if os_family() == "windows" { `Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ'` } else { `date -u '+%Y-%m-%dT%H:%M:%SZ'` }
ldflags_common := "-X github.com/Alia5/EnshroudedMixedInput/meta.Version=" + version + " -X github.com/Alia5/EnshroudedMixedInput/meta.Commit=" + commit + " -X github.com/Alia5/EnshroudedMixedInput/meta.Date=" + build_time
ldflags_release := "-s -w " + ldflags_common
extension_path := join(dist_dir, binary_name + ".dll")
header_path := join(dist_dir, binary_name + ".h")
go_licenses_cmd := "go run github.com/google/go-licenses/v2@v2.0.1"
licenses_template := "licenses.tpl"
licenses_template_work := if os_family() == "windows" { join(env_var_or_default("TEMP", "."), "emi-licenses.rendered.tpl") } else { "/tmp/emi-licenses.rendered.tpl" }
licenses_ignore := "github.com/Alia5/EnshroudedMixedInput"
licenses_out := join(dist_dir, "licenses.txt")

export GOOS := "windows"
export GOARCH := "amd64"
export CGO_ENABLED := "1"

default:
    just --list

build-loader type="Debug":
    {{ mkdir_p }} {{ dist_dir }}
    cmake -S {{ loader_dir }} -B {{ loader_dir }}/build "-DEMI_VERSION={{ version }}" "-DEMI_FIXED_VERSION={{ fixed_version }}"
    cmake --build {{ loader_dir }}/build --config {{ type }}

win-resource:
    goversioninfo {{ versioninfo_flags }} -description "EnshroudedMixedInput" -o emi/extension/resource_windows_amd64.syso -original-name {{ binary_name }}.dll versioninfo.json

build-emi type="Debug": win-resource
    {{ mkdir_p }} {{ dist_dir }}
    go build -buildmode=c-shared -trimpath -ldflags "{{ if type == "Release" { ldflags_release } else { ldflags_common } }}" -o {{ extension_path }} {{ extension_pkg }}
    {{ if os_family() == "windows" { "if (Test-Path " + header_path + ") { Remove-Item " + header_path + " }" } else { "rm -f " + header_path } }}

[arg("type", pattern="Debug|Release")]
build type="Debug": (build-emi type) (build-loader type)

# Copy the build into the game directory (EMI_GAME_DIR).
[windows]
deploy: build
    Copy-Item -Force {{ join(dist_dir, "dinput8.dll") }}, {{ extension_path }} "{{ game_dir }}"

# Remove the build from the game directory (EMI_GAME_DIR).
[windows]
undeploy:
    Remove-Item -Force -ErrorAction 0 "{{ game_dir }}\dinput8.dll", "{{ game_dir }}\{{ binary_name }}.dll"

test:
    go test -count=1 -v ./...

fmt:
    go fmt ./...

lint:
    golangci-lint run ./...

[windows]
licenses:
    {{ mkdir_p }} {{ dist_dir }} | Out-Null; $template = (Get-Content {{ licenses_template }} -Raw -Encoding utf8).Replace('VERSION_PLACEHOLDER', '{{ version }}'); [System.IO.File]::WriteAllText("{{ licenses_template_work }}", $template, [System.Text.UTF8Encoding]::new($false)); {{ go_licenses_cmd }} report {{ extension_pkg }} --ignore {{ licenses_ignore }} --template {{ licenses_template_work }} | Set-Content -Encoding utf8 {{ licenses_out }}; Remove-Item -Force {{ licenses_template_work }} -ErrorAction SilentlyContinue

[unix]
licenses:
    {{ mkdir_p }} {{ dist_dir }} && sed "s/VERSION_PLACEHOLDER/{{ version }}/g" {{ licenses_template }} > {{ licenses_template_work }} && {{ go_licenses_cmd }} report {{ extension_pkg }} --ignore {{ licenses_ignore }} --template {{ licenses_template_work }} > {{ licenses_out }} && rm -f {{ licenses_template_work }}

clean:
    -{{ rm_rf }} {{ dist_dir }}
    -{{ rm_rf }} {{ loader_dir }}/build
    go clean
