#!/usr/bin/env pwsh
# End-to-end test for run.bat on Windows, mirroring the POSIX MCP handshake
# test in test_run.sh. Exercises the first-install code path: no binary in
# bin/, a stub curl.bat shadows real curl via PATH, the "download" copies a
# cross-compiled mock MCP server into place, and a real JSON-RPC initialize
# request is piped into `run.bat stdio` with the response asserted.

$ErrorActionPreference = 'Stop'

$PASS = 0
$FAIL = 0

function Pass($msg) { Write-Host "  PASS: $msg"; $script:PASS++ }
function Fail($msg) { Write-Host "  FAIL: $msg"; $script:FAIL++ }

Write-Host "=== stdio first-install MCP handshake test (run.bat) ==="

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot  = (Resolve-Path (Join-Path $ScriptDir '..')).Path

$TmpRoot     = (New-Item -ItemType Directory -Path (Join-Path $env:TEMP "lumen-stdio-$([guid]::NewGuid().ToString('N'))")).FullName
$FakeCurlDir = (New-Item -ItemType Directory -Path (Join-Path $env:TEMP "fakecurl-$([guid]::NewGuid().ToString('N'))")).FullName
$MockBinDir  = (New-Item -ItemType Directory -Path (Join-Path $env:TEMP "mockbin-$([guid]::NewGuid().ToString('N'))")).FullName

$origPath = $env:PATH
$buildOK  = $false
$proc     = $null

try {
    $arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'arm64' } else { 'amd64' }

    # Build the mock MCP server — pure Go, no CGO.
    $mockBin = Join-Path $MockBinDir 'mock_lumen.exe'
    $env:CGO_ENABLED = '0'
    Push-Location $RepoRoot
    try {
        $buildOutput = & go build -o $mockBin ./scripts/testdata/mock_mcp_server 2>&1
        if ($LASTEXITCODE -eq 0) {
            $buildOK = $true
        } else {
            Fail "could not build mock MCP server (exit $LASTEXITCODE)"
            $buildOutput | ForEach-Object { Write-Host "          $_" }
        }
    } finally {
        Pop-Location
    }

    if ($buildOK) {
        # Minimal plugin root: manifest only.
        $manifest = '{' + "`n" + '  ".": "0.0.1"' + "`n" + '}' + "`n"
        [IO.File]::WriteAllText((Join-Path $TmpRoot '.release-please-manifest.json'), $manifest, [Text.Encoding]::ASCII)
        New-Item -ItemType Directory -Path (Join-Path $TmpRoot 'bin') | Out-Null

        # Stub curl: curl.bat parses -o <target> and copies the prebuilt mock in.
        # cmd.exe's PATHEXT search is per-directory: our fake dir is prepended
        # to PATH and contains only curl.bat, so it wins regardless of PATHEXT
        # ordering (each directory is tried fully before moving to the next).
        $curlStub = @'
@echo off
setlocal enabledelayedexpansion
echo %*>>"%LUMEN_CURL_ARGS_LOG%"
:loop
if "%~1"=="" goto done
if "%~1"=="-o" (
  copy /Y "%LUMEN_MOCK_BINARY%" "%~2" >nul
  shift
  shift
  goto loop
)
shift
goto loop
:done
exit /b 0
'@
        [IO.File]::WriteAllText((Join-Path $FakeCurlDir 'curl.bat'), $curlStub, [Text.Encoding]::ASCII)

        function Invoke-RunBatScenario {
            param(
                [string] $Name,
                [string] $ExpectedRepo,
                [string] $PassMessage,
                [string] $ReleaseRepoOverride = '',
                [string] $OriginUrl = ''
            )

            $pluginRoot = (New-Item -ItemType Directory -Path (Join-Path $TmpRoot $Name)).FullName
            $curlArgsLog = Join-Path $pluginRoot 'curl-args.txt'
            $expectedBinary = Join-Path $pluginRoot "bin\lumen-windows-$arch.exe"

            # Minimal plugin root: manifest only.
            $manifest = '{' + "`n" + '  ".": "0.0.1"' + "`n" + '}' + "`n"
            [IO.File]::WriteAllText((Join-Path $pluginRoot '.release-please-manifest.json'), $manifest, [Text.Encoding]::ASCII)
            New-Item -ItemType Directory -Path (Join-Path $pluginRoot 'bin') | Out-Null

            if ($OriginUrl) {
                $gitOutput = & git -C $pluginRoot init 2>&1
                if ($LASTEXITCODE -ne 0) {
                    Fail "could not initialise plugin root git repo for $Name"
                    $gitOutput | ForEach-Object { Write-Host "          $_" }
                    return
                }

                $gitOutput = & git -C $pluginRoot remote add origin $OriginUrl 2>&1
                if ($LASTEXITCODE -ne 0) {
                    Fail "could not add origin remote for $Name"
                    $gitOutput | ForEach-Object { Write-Host "          $_" }
                    return
                }
            }

            # Launch run.bat via System.Diagnostics.Process for reliable exit-code
            # propagation and explicit stdin/stdout/stderr wiring. Start-Process
            # with -RedirectStandardInput is unreliable here.
            $runBat  = Join-Path $ScriptDir 'run.bat'
            $initReq = '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"launcher-e2e","version":"1.0"}}}'

            $psi = New-Object System.Diagnostics.ProcessStartInfo
            $psi.FileName = 'cmd.exe'
            $psi.Arguments = "/c `"$runBat`" stdio"
            $psi.UseShellExecute = $false
            $psi.RedirectStandardInput  = $true
            $psi.RedirectStandardOutput = $true
            $psi.RedirectStandardError  = $true
            $psi.CreateNoWindow = $true
            $psi.WorkingDirectory = $RepoRoot
            $psi.Environment['CLAUDE_PLUGIN_ROOT'] = $pluginRoot
            $psi.Environment['LUMEN_MOCK_BINARY']  = $mockBin
            $psi.Environment['LUMEN_CURL_ARGS_LOG'] = $curlArgsLog
            $psi.Environment['PATH'] = "$FakeCurlDir;$origPath"
            $psi.Environment.Remove('LUMEN_RELEASE_REPO') | Out-Null
            if ($ReleaseRepoOverride) {
                $psi.Environment['LUMEN_RELEASE_REPO'] = $ReleaseRepoOverride
            }

            $script:proc = [System.Diagnostics.Process]::Start($psi)

            # If run.bat fast-exits in stdio mode, its stdin pipe is already
            # closed by the time we try to write — that IS the #125 symptom,
            # so swallow the broken-pipe exception and let the exit-code check
            # below produce the real diagnostic.
            try { $script:proc.StandardInput.WriteLine($initReq) } catch { }
            try { $script:proc.StandardInput.Close() } catch { }

            $stdout = $script:proc.StandardOutput.ReadToEnd()
            $stderr = $script:proc.StandardError.ReadToEnd()
            if (-not $script:proc.WaitForExit(60000)) {
                $script:proc.Kill()
                Fail "run.bat stdio did not exit within 60s for $Name"
            } else {
                $exitCode = $script:proc.ExitCode

                Write-Host "    launcher exit code ($Name): $exitCode"
                if ($stderr) {
                    Write-Host "    launcher stderr ($Name):"
                    ($stderr -split "`r?`n") | ForEach-Object { if ($_) { Write-Host "      $_" } }
                }

                $expectedUrl = [regex]::Escape("https://github.com/$ExpectedRepo/releases/download/")
                if ($exitCode -ne 0) {
                    Fail "run.bat stdio exited $exitCode for $Name - MCP server would be dead for the session"
                } elseif (-not (Test-Path $expectedBinary)) {
                    Fail "run.bat stdio did not place artefact at $expectedBinary"
                } elseif (-not (Test-Path $curlArgsLog)) {
                    Fail "fake curl did not capture launcher download arguments for $Name"
                } elseif ((Get-Content -Raw $curlArgsLog) -notmatch $expectedUrl) {
                    Fail "run.bat stdio should use $ExpectedRepo for download URL in $Name"
                    Write-Host "        curl args:"
                    (Get-Content $curlArgsLog) | ForEach-Object { Write-Host "          $_" }
                } elseif ($stdout -notmatch '"jsonrpc":"2\.0"') {
                    Fail "MCP initialize produced no JSON-RPC 2.0 response on stdout for $Name"
                    Write-Host "        stdout:"
                    ($stdout -split "`r?`n") | ForEach-Object { Write-Host "          $_" }
                } elseif ($stdout -notmatch '"name":"mock-lumen"') {
                    Fail "MCP response did not come from the exec'd mock for $Name - run.bat may be swallowing stdout"
                    Write-Host "        stdout:"
                    ($stdout -split "`r?`n") | ForEach-Object { Write-Host "          $_" }
                } else {
                    Pass $PassMessage
                }
            }
        }

        Invoke-RunBatScenario `
            -Name 'env-override' `
            -ExpectedRepo 'def324/lumen' `
            -ReleaseRepoOverride 'def324/lumen' `
            -PassMessage 'run.bat stdio uses LUMEN_RELEASE_REPO for first-install download'

        Invoke-RunBatScenario `
            -Name 'origin-remote' `
            -ExpectedRepo 'def324/lumen' `
            -OriginUrl 'https://github.com/def324/lumen.git' `
            -PassMessage 'run.bat stdio derives first-install download repo from git origin'

        Invoke-RunBatScenario `
            -Name 'invalid-origin-fallback' `
            -ExpectedRepo 'ory/lumen' `
            -OriginUrl 'https://github.com/def324/lumen/archive/main.tar.gz' `
            -PassMessage 'run.bat stdio ignores invalid GitHub origin shapes'

        Invoke-RunBatScenario `
            -Name 'invalid-owner-fallback' `
            -ExpectedRepo 'ory/lumen' `
            -OriginUrl 'https://github.com/def_324/lumen.git' `
            -PassMessage 'run.bat stdio ignores invalid GitHub origin owners'

        Invoke-RunBatScenario `
            -Name 'long-owner-fallback' `
            -ExpectedRepo 'ory/lumen' `
            -OriginUrl 'https://github.com/abcdefghijklmnopqrstuvwxyzabcdefghijklmn/lumen.git' `
            -PassMessage 'run.bat stdio ignores GitHub origin owners over 39 characters'
    }
} finally {
    $env:PATH = $origPath
    if ($proc -and -not $proc.HasExited) { try { $proc.Kill() } catch {} }
    Remove-Item -Recurse -Force $TmpRoot, $FakeCurlDir, $MockBinDir -ErrorAction SilentlyContinue
}

Write-Host ''
Write-Host '=== summary ==='
Write-Host "  passed: $PASS"
Write-Host "  failed: $FAIL"
if ($FAIL -gt 0) { exit 1 }
exit 0
