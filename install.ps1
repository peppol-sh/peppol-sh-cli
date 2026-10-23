# Installs the peppol.sh CLI (peppol-sh.exe) on Windows. Windows PowerShell 5.1
# or PowerShell 7.
#
#   irm https://raw.githubusercontent.com/peppol-sh/peppol-sh-cli/main/install.ps1 | iex
#
# Environment (all optional):
#   PEPPOL_SH_VERSION        version to install, e.g. 0.1.0 (default: the latest release)
#   PEPPOL_SH_INSTALL_DIR    target directory (default: %LOCALAPPDATA%\peppol-sh\bin)
#   PEPPOL_SH_DOWNLOAD_BASE  for tests and mirrors: URL of a directory with the
#                            archives and checksums.txt (default: the GitHub
#                            release). Both files come from there, so use only
#                            a trusted https URL.
#
# The archive is checked against the sha256 in checksums.txt before install.
# The script throws instead of calling `exit`: under `iex`, `exit` would close
# the caller's PowerShell window.

function Install-PeppolSh {
    $ErrorActionPreference = 'Stop'
    # Invoke-WebRequest in Windows PowerShell 5.1 is very slow with the progress bar.
    $ProgressPreference = 'SilentlyContinue'
    # Windows PowerShell 5.1 does not offer TLS 1.2 by default; GitHub requires it.
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

    $repo = 'peppol-sh/peppol-sh-cli'

    # A 32-bit PowerShell on 64-bit Windows reports x86 here and the real
    # architecture in PROCESSOR_ARCHITEW6432.
    $cpu = $env:PROCESSOR_ARCHITEW6432
    if (-not $cpu) { $cpu = $env:PROCESSOR_ARCHITECTURE }
    switch ($cpu) {
        'AMD64' { $arch = 'amd64' }
        'ARM64' { $arch = 'arm64' }
        default { throw "peppol-sh: unsupported architecture '$cpu' (supported: AMD64, ARM64)" }
    }
    $archive = "peppol-sh_windows_$arch.zip"

    $version = "$env:PEPPOL_SH_VERSION".TrimStart('v')
    if ($env:PEPPOL_SH_DOWNLOAD_BASE) {
        $base = $env:PEPPOL_SH_DOWNLOAD_BASE.TrimEnd('/')
    } elseif ($version) {
        $base = "https://github.com/$repo/releases/download/v$version"
    } else {
        $base = "https://github.com/$repo/releases/latest/download"
    }

    $installDir = $env:PEPPOL_SH_INSTALL_DIR
    if (-not $installDir) { $installDir = Join-Path $env:LOCALAPPDATA 'peppol-sh\bin' }

    $tmp = Join-Path ([IO.Path]::GetTempPath()) ('peppol-sh-' + [Guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $tmp | Out-Null
    try {
        $zip = Join-Path $tmp $archive
        $sums = Join-Path $tmp 'checksums.txt'
        Write-Host "Downloading $base/$archive"
        Invoke-WebRequest -UseBasicParsing -Uri "$base/$archive" -OutFile $zip
        Invoke-WebRequest -UseBasicParsing -Uri "$base/checksums.txt" -OutFile $sums

        $expected = $null
        foreach ($line in Get-Content -Path $sums) {
            $fields = $line -split '\s+'
            if ($fields.Count -ge 2 -and $fields[1] -eq $archive) { $expected = $fields[0] }
        }
        if (-not $expected) { throw "peppol-sh: checksums.txt has no entry for $archive" }
        $actual = (Get-FileHash -Algorithm SHA256 -Path $zip).Hash
        # -ne is case-insensitive: Get-FileHash returns upper case, checksums.txt lower case.
        if ($actual -ne $expected) {
            throw "peppol-sh: checksum mismatch for $archive (expected $expected, got $actual)"
        }

        # Extract to the temp folder and copy only the binary: the archive also
        # holds LICENSE, README.md and CHANGELOG.md, which do not belong in a
        # bin folder.
        $extract = Join-Path $tmp 'extract'
        Expand-Archive -Path $zip -DestinationPath $extract -Force
        $exe = Join-Path $extract 'peppol-sh.exe'
        if (-not (Test-Path -LiteralPath $exe)) { throw "peppol-sh: $archive does not contain peppol-sh.exe" }

        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
        $target = Join-Path $installDir 'peppol-sh.exe'
        Copy-Item -LiteralPath $exe -Destination $target -Force
        Write-Host "Installed $target"
    } finally {
        Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
    }

    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $entries = @()
    if ($userPath) { $entries = $userPath -split ';' | Where-Object { $_ } }
    $normalized = $installDir.TrimEnd('\')
    if (-not ($entries | Where-Object { $_.TrimEnd('\') -eq $normalized })) {
        [Environment]::SetEnvironmentVariable('Path', (@($entries) + $installDir) -join ';', 'User')
        Write-Host "Added $installDir to your user PATH. Open a new terminal to use peppol-sh."
    }
    if (-not (($env:Path -split ';') | Where-Object { $_.TrimEnd('\') -eq $normalized })) {
        $env:Path = "$env:Path;$installDir"
    }

    & $target version
}

Install-PeppolSh
