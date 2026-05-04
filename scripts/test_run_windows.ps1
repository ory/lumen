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
$CurlArgsLog = Join-Path $TmpRoot 'curl-args.txt'

$origPath = $env:PATH
$buildOK  = $false
$proc     = $null

try {
    $arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'arm64' } else { 'amd64' }
    $expectedBinary = Join-Path $TmpRoot "bin\lumen-windows-$arch.exe"

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
        $psi.Environment['CLAUDE_PLUGIN_ROOT'] = $TmpRoot
        $psi.Environment['LUMEN_MOCK_BINARY']  = $mockBin
        $psi.Environment['LUMEN_RELEASE_REPO'] = 'def324/lumen'
        $psi.Environment['LUMEN_CURL_ARGS_LOG'] = $CurlArgsLog
        $psi.Environment['PATH'] = "$FakeCurlDir;$origPath"

        $proc = [System.Diagnostics.Process]::Start($psi)

        # If run.bat fast-exits in stdio mode, its stdin pipe is already
        # closed by the time we try to write — that IS the #125 symptom,
        # so swallow the broken-pipe exception and let the exit-code check
        # below produce the real diagnostic.
        try { $proc.StandardInput.WriteLine($initReq) } catch { }
        try { $proc.StandardInput.Close() } catch { }

        $stdout = $proc.StandardOutput.ReadToEnd()
        $stderr = $proc.StandardError.ReadToEnd()
        if (-not $proc.WaitForExit(60000)) {
            $proc.Kill()
            Fail "run.bat stdio did not exit within 60s"
        } else {
            $exitCode = $proc.ExitCode

            Write-Host "    launcher exit code: $exitCode"
            if ($stderr) {
                Write-Host "    launcher stderr:"
                ($stderr -split "`r?`n") | ForEach-Object { if ($_) { Write-Host "      $_" } }
            }

            if ($exitCode -ne 0) {
                Fail "run.bat stdio exited $exitCode — MCP server would be dead for the session"
            } elseif (-not (Test-Path $expectedBinary)) {
                Fail "run.bat stdio did not place artefact at $expectedBinary"
            } elseif (-not (Test-Path $CurlArgsLog)) {
                Fail "fake curl did not capture launcher download arguments"
            } elseif ((Get-Content -Raw $CurlArgsLog) -notmatch 'https://github\.com/def324/lumen/releases/download/') {
                Fail "run.bat stdio should use LUMEN_RELEASE_REPO for download URL"
                Write-Host "        curl args:"
                (Get-Content $CurlArgsLog) | ForEach-Object { Write-Host "          $_" }
            } elseif ($stdout -notmatch '"jsonrpc":"2\.0"') {
                Fail "MCP initialize produced no JSON-RPC 2.0 response on stdout"
                Write-Host "        stdout:"
                ($stdout -split "`r?`n") | ForEach-Object { Write-Host "          $_" }
            } elseif ($stdout -notmatch '"name":"mock-lumen"') {
                Fail "MCP response did not come from the exec'd mock — run.bat may be swallowing stdout"
                Write-Host "        stdout:"
                ($stdout -split "`r?`n") | ForEach-Object { Write-Host "          $_" }
            } else {
                Pass "run.bat stdio downloads, execs, and brokers MCP initialize on first install"
            }
        }
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
