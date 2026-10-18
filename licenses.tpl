{{ $emiVersion := "VERSION_PLACEHOLDER" }}

# github.com/Alia5/EnshroudedMixedInput

* Name: github.com/Alia5/EnshroudedMixedInput
* Version: {{ $emiVersion }}
* License: [GPL-3.0](https://github.com/Alia5/EnshroudedMixedInput/blob/main/LICENSE.txt)

EnshroudedMixedInput

Copyright (C) 2026 Peter Repukat

This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public
License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later
version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

{{ range . }}
## {{ .Name }}

* Name: {{ .Name }}
* Version: {{ .Version }}
* License: [{{ .LicenseName }}]({{ .LicenseURL }})
{{ end }}
