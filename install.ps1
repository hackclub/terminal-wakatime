param(
    [string]$Version = 'latest',
    [switch]$SkipProfile,
    [switch]$Force
)

$ErrorActionPreference = 'Stop'

$Repo = 'hackclub/terminal-wakatime'
$BinaryBaseName = 'terminal-wakatime'
$InstallDir = Join-Path $HOME '.wakatime'
$ProfileMarker = '# terminal-wakatime setup'

function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor Green
}

function Write-WarningMsg {
    param([string]$Message)
    Write-Host "[WARNING] $Message" -ForegroundColor Yellow
}

function Get-PlatformAsset {
    $os = if ($IsWindows) {
        'windows'
    } elseif ($IsMacOS) {
        'darwin'
    } elseif ($IsLinux) {
        'linux'
    } else {
        throw "Unsupported operating system for this installer."
    }

    $runtimeArch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
    $arch = switch ($runtimeArch) {
        'x64' { 'amd64' }
        'arm64' { 'arm64' }
        default { throw "Unsupported architecture: $runtimeArch" }
    }

    if ($os -eq 'windows') {
        return "$os-$arch.exe"
    }

    return "$os-$arch"
}

function Get-LatestVersion {
    Write-Info 'Fetching latest release information...'
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -Headers @{ 'User-Agent' = 'terminal-wakatime-install-ps1' }
    if (-not $release.tag_name) {
        throw 'Failed to fetch latest version from GitHub API.'
    }
    return $release.tag_name
}

function Ensure-InstallDir {
    if (-not (Test-Path -LiteralPath $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }
}

function Add-ToCurrentPath {
    $parts = $env:PATH -split [IO.Path]::PathSeparator
    if ($parts -notcontains $InstallDir) {
        $env:PATH = "$InstallDir$([IO.Path]::PathSeparator)$env:PATH"
    }
}

function Test-ParseablePowerShell {
    param([string]$ScriptText)

    if ([string]::IsNullOrWhiteSpace($ScriptText)) {
        return $false
    }

    try {
        [ScriptBlock]::Create($ScriptText) | Out-Null
        return $true
    } catch {
        return $false
    }
}

function Get-SafeInitBlock {
    return @"
`$twBinary = Join-Path `$twInstallDir 'terminal-wakatime'
if (`$IsWindows) { `$twBinary = "`$twBinary.exe" }
if (Test-Path -LiteralPath `$twBinary) {
    `$twHooks = & `$twBinary init powershell 2>`$null
    `$twHooksText = [string]::Join([Environment]::NewLine, @(`$twHooks))
    `$twCanParseHooks = `$false
    if (-not [string]::IsNullOrWhiteSpace(`$twHooksText)) {
        try {
            [ScriptBlock]::Create(`$twHooksText) | Out-Null
            `$twCanParseHooks = `$true
        } catch {
            `$twCanParseHooks = `$false
        }
    }
    if (`$twCanParseHooks) {
        Invoke-Expression `$twHooksText
    }
}
"@
}

function Get-ProfileIntegrationBlock {
    param([string]$InstallDirPath)

    $safeInitBlock = Get-SafeInitBlock

    return @"

$ProfileMarker
`$twInstallDir = '$InstallDirPath'
if (-not ((`$env:PATH -split [IO.Path]::PathSeparator) -contains `$twInstallDir)) {
    `$env:PATH = "`${twInstallDir}$([IO.Path]::PathSeparator)`$env:PATH"
}
$safeInitBlock
"@
}

function Add-ProfileIntegration {
    $profilePath = $PROFILE.CurrentUserCurrentHost
    $profileDir = Split-Path -Parent $profilePath

    if (-not (Test-Path -LiteralPath $profileDir)) {
        New-Item -ItemType Directory -Path $profileDir -Force | Out-Null
    }

    if (-not (Test-Path -LiteralPath $profilePath)) {
        New-Item -ItemType File -Path $profilePath -Force | Out-Null
    }

    $existing = Get-Content -LiteralPath $profilePath -Raw -ErrorAction SilentlyContinue
    $updated = $existing

    # Migrate legacy invalid interpolation: "$twInstallDir:$env:PATH"
    $updated = $updated.Replace(
        '$env:PATH = "$twInstallDir:$env:PATH"',
        '$env:PATH = "${twInstallDir}$([IO.Path]::PathSeparator)$env:PATH"'
    )

    # Migrate legacy one-line init and malformed binary name
    $updated = $updated.Replace(
        'terminal-wakatime init powershell | Invoke-Expression',
        (Get-SafeInitBlock).Trim()
    )
    $updated = $updated.Replace(
        "`$twBinary = Join-Path `$twInstallDir ''terminal-wakatime''",
        "`$twBinary = Join-Path `$twInstallDir 'terminal-wakatime'"
    )

    # Migrate direct Invoke-Expression on hook text to parse-checked version
    $legacyEvalSnippet = @'
$twHooksText = [string]::Join([Environment]::NewLine, @($twHooks))
if (-not [string]::IsNullOrWhiteSpace($twHooksText)) {
    Invoke-Expression $twHooksText
}
'@
    $guardedEvalSnippet = @'
$twHooksText = [string]::Join([Environment]::NewLine, @($twHooks))
$twCanParseHooks = $false
if (-not [string]::IsNullOrWhiteSpace($twHooksText)) {
    try {
        [ScriptBlock]::Create($twHooksText) | Out-Null
        $twCanParseHooks = $true
    } catch {
        $twCanParseHooks = $false
    }
}
if ($twCanParseHooks) {
    Invoke-Expression $twHooksText
}
'@
    $updated = $updated.Replace($legacyEvalSnippet.Trim(), $guardedEvalSnippet.Trim())

    if ($updated -ne $existing) {
        Set-Content -LiteralPath $profilePath -Value $updated
        Write-Success "Updated existing terminal-wakatime profile entries: $profilePath"
        $existing = $updated
    }

    if ($existing -and $existing.Contains("`$twHooks = & `$twBinary init powershell")) {
        Write-WarningMsg "PowerShell profile already contains terminal-wakatime integration: $profilePath"
        return
    }

    Add-Content -LiteralPath $profilePath -Value (Get-ProfileIntegrationBlock -InstallDirPath $InstallDir)
    Write-Success "Added terminal-wakatime integration to profile: $profilePath"
}

try {
    $assetPlatform = Get-PlatformAsset
    Write-Info "Detected platform: $assetPlatform"

    $resolvedVersion = if ($Version -eq 'latest') { Get-LatestVersion } else { $Version }
    Write-Info "Using version: $resolvedVersion"

    $downloadUrl = "https://github.com/$Repo/releases/download/$resolvedVersion/$BinaryBaseName-$assetPlatform"
    $binaryName = if ($IsWindows) { "$BinaryBaseName.exe" } else { $BinaryBaseName }
    $targetPath = Join-Path $InstallDir $binaryName

    Ensure-InstallDir

    if ((Test-Path -LiteralPath $targetPath) -and -not $Force) {
        Write-WarningMsg "Binary already exists at $targetPath. Re-run with -Force to overwrite."
    } else {
        $tempPath = Join-Path ([IO.Path]::GetTempPath()) "$binaryName.$([Guid]::NewGuid().ToString('N'))"
        Write-Info "Downloading from: $downloadUrl"
        Invoke-WebRequest -Uri $downloadUrl -OutFile $tempPath

        Move-Item -Path $tempPath -Destination $targetPath -Force
        if (-not $IsWindows) {
            & chmod +x $targetPath
        }

        Write-Success "Installed binary to: $targetPath"
    }

    Add-ToCurrentPath

    if (-not $SkipProfile) {
        Add-ProfileIntegration
    }

    if (Test-Path -LiteralPath $targetPath) {
        try {
            $sessionHooks = & $targetPath init powershell 2>$null
            $sessionHooksText = [string]::Join([Environment]::NewLine, @($sessionHooks))

            if (Test-ParseablePowerShell $sessionHooksText) {
                Invoke-Expression $sessionHooksText
                Write-Success 'Initialized terminal-wakatime hooks for this session.'
            } else {
                Write-WarningMsg 'Binary returned empty or non-PowerShell hooks; skipping session initialization.'
            }
        } catch {
            Write-WarningMsg "Installed successfully, but failed to initialize hooks in current session: $($_.Exception.Message)"
        }
    }

    Write-Host ''
    Write-Success 'Installation complete!'
    Write-Info 'Next steps:'
    Write-Host '  1. Run: terminal-wakatime config --key YOUR_WAKATIME_KEY'
    Write-Host '  2. Restart your PowerShell session (or run: . $PROFILE)'
}
catch {
    Write-Host "[ERROR] $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
