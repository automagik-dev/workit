# ---------------------------------------------------------------------------
# workit installer for Windows
#
# Downloads and installs the wk binary and workit plugin from GitHub Releases.
#
# Usage:
#   irm https://raw.githubusercontent.com/automagik-dev/workit/main/install.ps1 | iex
#   .\install.ps1 [-Force] [-Version VERSION] [-SkipPrereqs] [-Help]
#
# ---------------------------------------------------------------------------

#Requires -Version 5.1

param(
    [switch]$Force,
    [string]$Version,
    [switch]$SkipPrereqs,
    [switch]$Help
)

$ErrorActionPreference = "Stop"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
function Write-Info  { param([string]$Msg) Write-Host "[INFO]  $Msg" -ForegroundColor Blue }
function Write-Ok    { param([string]$Msg) Write-Host "[OK]    $Msg" -ForegroundColor Green }
function Write-Warn  { param([string]$Msg) Write-Host "[WARN]  $Msg" -ForegroundColor Yellow }
function Write-Fail  { param([string]$Msg) Write-Host "[ERROR] $Msg" -ForegroundColor Red; exit 1 }

# ---------------------------------------------------------------------------
# Help
# ---------------------------------------------------------------------------
if ($Help) {
    Write-Host @"
Usage: .\install.ps1 [OPTIONS]

Downloads and installs workit (wk binary + plugin) from GitHub Releases.

Options:
  -Force              Skip confirmation prompts and overwrite existing install
  -Version VERSION    Install a specific version (e.g. 2.260227.5)
                      Default: latest release
  -SkipPrereqs        Skip prerequisite installation (LibreOffice, Python, lxml)
  -Help               Show this help message and exit

Examples:
  .\install.ps1
  .\install.ps1 -Force
  .\install.ps1 -Version 2.260227.5
  irm https://raw.githubusercontent.com/automagik-dev/workit/main/install.ps1 | iex
"@
    exit 0
}

# ---------------------------------------------------------------------------
# PowerShell version check
# ---------------------------------------------------------------------------
if ($PSVersionTable.PSVersion.Major -lt 5) {
    Write-Error "PowerShell 5.1 or later is required. Current version: $($PSVersionTable.PSVersion). Please update PowerShell."
    exit 1
}

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------
$GitHubRepo   = "automagik-dev/workit"
$GitHubAPI    = "https://api.github.com/repos/$GitHubRepo/releases/latest"
$ReleaseBase  = "https://github.com/$GitHubRepo/releases/download"
$InstallDir   = Join-Path $env:LOCALAPPDATA "workit\bin"
$PluginDir    = Join-Path $env:USERPROFILE ".workit\plugin"

# ---------------------------------------------------------------------------
# Temp dir + cleanup
# ---------------------------------------------------------------------------
$TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) "workit_install_$([System.Guid]::NewGuid().ToString('N').Substring(0,8))"
New-Item -ItemType Directory -Path $TmpDir -Force | Out-Null

function Invoke-Cleanup {
    if (Test-Path $TmpDir) {
        Remove-Item -Recurse -Force $TmpDir -ErrorAction SilentlyContinue
    }
}

try {

# ---------------------------------------------------------------------------
# Architecture detection
# ---------------------------------------------------------------------------
$Arch = $null
try {
    $OsArch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
    switch ($OsArch) {
        ([System.Runtime.InteropServices.Architecture]::X64)   { $Arch = "amd64" }
        ([System.Runtime.InteropServices.Architecture]::Arm64) { $Arch = "arm64" }
        default { Write-Fail "Unsupported architecture: $OsArch. Only amd64 and arm64 are supported." }
    }
} catch {
    # Fallback for older PowerShell
    switch ($env:PROCESSOR_ARCHITECTURE) {
        "AMD64" { $Arch = "amd64" }
        "ARM64" { $Arch = "arm64" }
        default { Write-Fail "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE. Only AMD64 and ARM64 are supported." }
    }
}

Write-Info "Detected platform: windows/$Arch"

# ---------------------------------------------------------------------------
# Version resolution
# ---------------------------------------------------------------------------
if (-not $Version) {
    Write-Info "Fetching latest release version..."
    try {
        $ReleaseInfo = Invoke-RestMethod -Uri $GitHubAPI -UseBasicParsing
        $Version = $ReleaseInfo.tag_name
    } catch {
        Write-Fail "Failed to fetch release info from $GitHubAPI : $_"
    }

    if (-not $Version) {
        Write-Fail "Could not parse version from GitHub API response."
    }
}

# Ensure Version has no v prefix, Tag has it
$Version = $Version -replace '^v', ''
$Tag     = "v$Version"

Write-Info "Installing workit $Tag"

# ---------------------------------------------------------------------------
# Binary download and install
# ---------------------------------------------------------------------------
$BinaryFilename = "workit_${Version}_windows_${Arch}.zip"
$BinaryUrl      = "$ReleaseBase/$Tag/$BinaryFilename"
$BinaryArchive  = Join-Path $TmpDir $BinaryFilename

Write-Info "Downloading binary: $BinaryUrl"
try {
    Invoke-WebRequest -Uri $BinaryUrl -OutFile $BinaryArchive -UseBasicParsing
} catch {
    Write-Fail "Failed to download binary from $BinaryUrl : $_"
}

# Download checksums
$ChecksumsUrl  = "$ReleaseBase/$Tag/checksums.txt"
$ChecksumsFile = Join-Path $TmpDir "checksums.txt"

Write-Info "Downloading checksums: $ChecksumsUrl"
try {
    Invoke-WebRequest -Uri $ChecksumsUrl -OutFile $ChecksumsFile -UseBasicParsing
} catch {
    Write-Fail "Failed to download checksums.txt from $ChecksumsUrl : $_"
}

# Verify checksum
function Test-Checksum {
    param(
        [string]$Filename,
        [string]$FilePath,
        [string]$ChecksumsPath
    )

    $lines = Get-Content $ChecksumsPath
    $entry = $lines | Where-Object { $_ -match "\s+$([regex]::Escape($Filename))$" } | Select-Object -First 1

    if (-not $entry) {
        Write-Fail "Checksum entry not found for $Filename in checksums.txt"
    }

    $expectedHash = ($entry -split '\s+')[0]
    $actualHash   = (Get-FileHash -Path $FilePath -Algorithm SHA256).Hash.ToLower()

    if ($actualHash -ne $expectedHash.ToLower()) {
        Write-Fail "Checksum mismatch for $Filename (expected: $expectedHash, got: $actualHash)"
    }
    Write-Ok "Checksum verified: $Filename"
}

if (Test-Path $ChecksumsFile) {
    Test-Checksum -Filename $BinaryFilename -FilePath $BinaryArchive -ChecksumsPath $ChecksumsFile
}

# Extract binary
Write-Info "Extracting binary..."
$ExtractDir = Join-Path $TmpDir "binary_extract"
Expand-Archive -Path $BinaryArchive -DestinationPath $ExtractDir -Force

# Find wk.exe
$WkBinary = Get-ChildItem -Path $ExtractDir -Filter "wk.exe" -Recurse -File | Select-Object -First 1
if (-not $WkBinary) {
    Write-Fail "Could not find 'wk.exe' in archive $BinaryFilename"
}

# Check for existing install
$Target     = Join-Path $InstallDir "wk.exe"
$SkipBinary = $false

if ((Test-Path $Target) -and -not $Force) {
    $ExistingVersion = "unknown"
    try { $ExistingVersion = & $Target --version 2>$null | Select-Object -First 1 } catch {}
    Write-Warn "Existing installation found at $Target"
    Write-Info "Installed version : $ExistingVersion"
    Write-Info "New version       : $Version"

    $reply = Read-Host "Overwrite? [y/N]"
    if ($reply -notmatch '^[Yy]') {
        Write-Info "Skipping binary overwrite. Continuing with plugin..."
        $SkipBinary = $true
    }
}

if (-not $SkipBinary) {
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }
    Copy-Item -Path $WkBinary.FullName -Destination $Target -Force
    Write-Ok "Binary installed: $Target"
} else {
    Write-Info "Binary unchanged at $Target"
}

# ---------------------------------------------------------------------------
# Plugin download and install
# ---------------------------------------------------------------------------
$PluginFilename = "workit-plugin_${Version}.tar.gz"
$PluginUrl      = "$ReleaseBase/$Tag/$PluginFilename"
$PluginArchive  = Join-Path $TmpDir $PluginFilename

Write-Info "Downloading plugin: $PluginUrl"
try {
    Invoke-WebRequest -Uri $PluginUrl -OutFile $PluginArchive -UseBasicParsing
} catch {
    Write-Fail "Failed to download plugin from $PluginUrl : $_"
}

if (Test-Path $ChecksumsFile) {
    Test-Checksum -Filename $PluginFilename -FilePath $PluginArchive -ChecksumsPath $ChecksumsFile
}

Write-Info "Installing plugin to $PluginDir..."

# Remove old plugin contents and recreate
if (Test-Path $PluginDir) {
    Remove-Item -Recurse -Force $PluginDir
}
New-Item -ItemType Directory -Path $PluginDir -Force | Out-Null

# Extract plugin tarball — requires tar (available on Windows 10+)
if (Get-Command tar -ErrorAction SilentlyContinue) {
    $PluginExtract = Join-Path $TmpDir "plugin_extract"
    New-Item -ItemType Directory -Path $PluginExtract -Force | Out-Null
    & tar -xzf $PluginArchive -C $PluginExtract 2>$null

    if ($LASTEXITCODE -ne 0) {
        Write-Warn "tar extraction failed. Plugin install skipped."
    } else {
        # Move extracted contents (inside workit/ subdirectory) to PluginDir
        $WorkitSubdir = Join-Path $PluginExtract "workit"
        if (Test-Path $WorkitSubdir) {
            Copy-Item -Path "$WorkitSubdir\*" -Destination $PluginDir -Recurse -Force
        } else {
            Copy-Item -Path "$PluginExtract\*" -Destination $PluginDir -Recurse -Force
        }
        Write-Ok "Plugin installed: $PluginDir"
    }
} else {
    Write-Warn "tar command not found. Plugin extraction skipped."
    Write-Warn "Please install tar or manually extract $PluginArchive to $PluginDir"
}

# ---------------------------------------------------------------------------
# Claude Code integration
# ---------------------------------------------------------------------------
if (Get-Command claude -ErrorAction SilentlyContinue) {
    # Register workit repo as a marketplace and install the plugin
    & claude plugin marketplace add https://github.com/automagik-dev/workit.git 2>$null
    try {
        & claude plugin install "workit@automagik-workit" 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Ok "Claude Code plugin installed: workit@automagik-workit"
        } else {
            throw "non-zero exit"
        }
    } catch {
        # Fallback: symlink for older Claude Code versions
        $ClaudePluginsDir = Join-Path $env:USERPROFILE ".claude\plugins"
        if (-not (Test-Path $ClaudePluginsDir)) {
            New-Item -ItemType Directory -Path $ClaudePluginsDir -Force | Out-Null
        }
        $SymlinkTarget = Join-Path $ClaudePluginsDir "workit"
        if (Test-Path $SymlinkTarget) {
            Remove-Item -Path $SymlinkTarget -Force -Recurse
        }
        # Try symbolic link, fall back to junction
        try {
            New-Item -ItemType SymbolicLink -Path $SymlinkTarget -Target $PluginDir -Force | Out-Null
        } catch {
            cmd /c mklink /J "$SymlinkTarget" "$PluginDir" 2>$null | Out-Null
        }
        Write-Ok "Claude Code plugin linked: $SymlinkTarget (fallback)"
    }
} else {
    $ClaudePluginsDir = Join-Path $env:USERPROFILE ".claude\plugins"
    if (-not (Test-Path $ClaudePluginsDir)) {
        New-Item -ItemType Directory -Path $ClaudePluginsDir -Force | Out-Null
    }
    $SymlinkTarget = Join-Path $ClaudePluginsDir "workit"
    if (Test-Path $SymlinkTarget) {
        Remove-Item -Path $SymlinkTarget -Force -Recurse
    }
    try {
        New-Item -ItemType SymbolicLink -Path $SymlinkTarget -Target $PluginDir -Force | Out-Null
    } catch {
        cmd /c mklink /J "$SymlinkTarget" "$PluginDir" 2>$null | Out-Null
    }
    Write-Ok "Claude Code plugin linked: $SymlinkTarget"
}

# ---------------------------------------------------------------------------
# PATH update
# ---------------------------------------------------------------------------
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -split ';' | Where-Object { $_ -eq $InstallDir }) {
    # Already in PATH
} else {
    $NewPath = if ($UserPath) { "$UserPath;$InstallDir" } else { $InstallDir }
    [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
    # Also update current session
    $env:Path = "$env:Path;$InstallDir"
    Write-Ok "Added $InstallDir to user PATH"
    Write-Info "Restart your terminal for PATH changes to take effect in new sessions."
}

# ---------------------------------------------------------------------------
# Prerequisites (unless -SkipPrereqs)
# ---------------------------------------------------------------------------
if (-not $SkipPrereqs) {
    Write-Info "Checking prerequisites..."

    # Detect package manager
    $PkgManager = $null
    if (Get-Command winget -ErrorAction SilentlyContinue) {
        $PkgManager = "winget"
    } elseif (Get-Command choco -ErrorAction SilentlyContinue) {
        $PkgManager = "choco"
    }

    # --- LibreOffice ---
    $HasSoffice = $false
    if (Get-Command soffice -ErrorAction SilentlyContinue) {
        $HasSoffice = $true
    } else {
        # Check common install paths
        $CommonPaths = @(
            "${env:ProgramFiles}\LibreOffice\program\soffice.exe",
            "${env:ProgramFiles(x86)}\LibreOffice\program\soffice.exe"
        )
        foreach ($p in $CommonPaths) {
            if (Test-Path $p) { $HasSoffice = $true; break }
        }
    }

    if ($HasSoffice) {
        Write-Ok "LibreOffice already installed"
    } else {
        if ($PkgManager -eq "winget") {
            Write-Info "Installing LibreOffice via winget..."
            winget install TheDocumentFoundation.LibreOffice --accept-package-agreements --accept-source-agreements
        } elseif ($PkgManager -eq "choco") {
            Write-Info "Installing LibreOffice via choco..."
            choco install libreoffice-fresh -y
        } else {
            Write-Warn "LibreOffice not found. Please install it manually from https://www.libreoffice.org/download/"
        }
    }

    # --- Python ---
    if (Get-Command python -ErrorAction SilentlyContinue) {
        Write-Ok "Python already installed"
    } else {
        if ($PkgManager -eq "winget") {
            Write-Info "Installing Python via winget..."
            winget install Python.Python.3.12 --accept-package-agreements --accept-source-agreements
        } elseif ($PkgManager -eq "choco") {
            Write-Info "Installing Python via choco..."
            choco install python3 -y
        } else {
            Write-Warn "Python not found. Please install it manually from https://www.python.org/downloads/"
        }
    }

    # --- lxml ---
    if (Get-Command python -ErrorAction SilentlyContinue) {
        $LxmlCheck = & python -c "import lxml" 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Ok "lxml already installed"
        } else {
            Write-Info "Installing lxml via pip..."
            & pip install lxml
        }
    }
} else {
    Write-Info "Skipping prerequisites check (-SkipPrereqs)"
}

# ---------------------------------------------------------------------------
# Summary banner
# ---------------------------------------------------------------------------
Write-Host ""
Write-Host ([char]0x2501 * 40)
if ($SkipBinary -and (Test-Path $Target)) {
    $DisplayVer = "unknown"
    try { $DisplayVer = & $Target --version 2>$null | Select-Object -First 1 } catch {}
    Write-Host "  plugin updated; wk unchanged ($DisplayVer)" -ForegroundColor Green
} else {
    Write-Host "  workit v$Version installed" -ForegroundColor Green
}
Write-Host ""
Write-Host "Binary:  $Target"
Write-Host "Plugin:  $PluginDir"
Write-Host "Claude:  $env:USERPROFILE\.claude\plugins\workit"
Write-Host "Skills:  loaded (Gmail, Calendar, Drive, Sheets, Docs, Slides, Chat, ...)"
Write-Host "Relay:   https://auth.automagik.dev (no GCP setup needed)"
Write-Host ([char]0x2501 * 40)

Write-Host ""
Write-Host "Next steps:"
Write-Host "  wk auth manage        -> connect your Google account" -ForegroundColor White
Write-Host "  wk --help             -> see all commands" -ForegroundColor White
Write-Host "  wk gmail search '...' -> try your first query" -ForegroundColor White
Write-Host ""
Write-Host "Docs: https://github.com/automagik-dev/workit"
Write-Host ""

} finally {
    Invoke-Cleanup
}
