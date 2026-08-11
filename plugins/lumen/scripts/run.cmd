@echo off
setlocal enabledelayedexpansion

:: Determine the package root from the host-specific variables before falling
:: back to the script location. Codex also injects a writable PLUGIN_DATA.
if defined CLAUDE_PLUGIN_ROOT (
  set "PACKAGE_ROOT=%CLAUDE_PLUGIN_ROOT%"
) else if defined CURSOR_PLUGIN_ROOT (
  set "PACKAGE_ROOT=%CURSOR_PLUGIN_ROOT%"
) else if defined PLUGIN_ROOT (
  set "PACKAGE_ROOT=%PLUGIN_ROOT%"
) else (
  set "PACKAGE_ROOT=%~dp0.."
)

if defined PLUGIN_DATA (
  set "BINARY_ROOT=%PLUGIN_DATA%"
) else (
  set "BINARY_ROOT=%PACKAGE_ROOT%"
)

:: Architecture detection
set "ARCH=amd64"
if "%PROCESSOR_ARCHITECTURE%"=="ARM64" set "ARCH=arm64"

:: Environment defaults
if not defined LUMEN_BACKEND set "LUMEN_BACKEND=ollama"
if not defined LUMEN_EMBED_MODEL set "LUMEN_EMBED_MODEL=ordis/jina-embeddings-v2-base-code"

:: Prefer a local development build or bundled binary. Downloads go to the
:: host-provided writable data directory when one exists.
set "BINARY="
if exist "%PACKAGE_ROOT%\bin\lumen.exe" set "BINARY=%PACKAGE_ROOT%\bin\lumen.exe"
if not defined BINARY if exist "%PACKAGE_ROOT%\bin\lumen-windows-%ARCH%.exe" set "BINARY=%PACKAGE_ROOT%\bin\lumen-windows-%ARCH%.exe"
if not defined BINARY if exist "%BINARY_ROOT%\bin\lumen-windows-%ARCH%.exe" set "BINARY=%BINARY_ROOT%\bin\lumen-windows-%ARCH%.exe"

:: Download on first run if binary is missing
if not defined BINARY (
  set "BINARY=%BINARY_ROOT%\bin\lumen-windows-%ARCH%.exe"
  set "REPO=ory/lumen"

  :: The repository launcher reads release-please metadata. The isolated Codex
  :: package carries the same version in its native plugin manifest.
  set "RELEASE_MANIFEST=%PACKAGE_ROOT%\.release-please-manifest.json"
  set "PLUGIN_MANIFEST=%PACKAGE_ROOT%\.codex-plugin\plugin.json"
  if exist "!RELEASE_MANIFEST!" (
    set "MANIFEST=!RELEASE_MANIFEST!"
    set "VERSION_FIELD=\"[.]\""
  ) else if exist "!PLUGIN_MANIFEST!" (
    set "MANIFEST=!PLUGIN_MANIFEST!"
    set "VERSION_FIELD=\"version\""
  ) else (
    echo Error: no release or plugin manifest found in %PACKAGE_ROOT% >&2
    exit /b 1
  )
  for /f "tokens=*" %%i in ('findstr /r "!VERSION_FIELD!" "!MANIFEST!"') do (
    for /f "tokens=2 delims=:" %%j in ("%%i") do (
      set "VERSION=v%%~j"
      set "VERSION=!VERSION: =!"
      set "VERSION=!VERSION:,=!"
      set "VERSION=!VERSION:"=!"
    )
  )

  if "!VERSION!"=="" (
    echo Error: could not read version from !MANIFEST! >&2
    exit /b 1
  )

  set "ASSET=lumen-!VERSION:~1!-windows-!ARCH!.exe"
  set "URL=https://github.com/!REPO!/releases/download/!VERSION!/!ASSET!"

  echo Downloading lumen !VERSION! for windows/!ARCH!... >&2
  if not exist "%BINARY_ROOT%\bin" mkdir "%BINARY_ROOT%\bin"

  call curl -sfL --max-time 300 --retry 3 --retry-delay 2 "!URL!" -o "!BINARY!"
  if errorlevel 1 (
    :: Fallback: manifest version not released yet — resolve latest from GitHub API
    echo Version !VERSION! not found, resolving latest release... >&2

    set "AUTH_HEADER="
    if defined GITHUB_TOKEN set "AUTH_HEADER=-H "Authorization: token %GITHUB_TOKEN%""

    set "TMPJSON=%TEMP%\lumen-latest.json"
    call curl -sfL !AUTH_HEADER! --max-time 30 --retry 2 --retry-delay 2 ^
      "https://api.github.com/repos/!REPO!/releases/latest" -o "!TMPJSON!"

    set "LATEST_TAG="
    for /f "tokens=2 delims=:" %%a in ('findstr /r "tag_name" "!TMPJSON!"') do (
      set "LATEST_TAG=%%~a"
      set "LATEST_TAG=!LATEST_TAG: =!"
      set "LATEST_TAG=!LATEST_TAG:,=!"
      set "LATEST_TAG=!LATEST_TAG:"=!"
    )
    del "!TMPJSON!" 2>nul

    if "!LATEST_TAG!"=="" (
      echo Error: could not resolve latest release from GitHub API >&2
      exit /b 1
    )
    echo !LATEST_TAG! | findstr /r "^v[0-9]" >nul 2>&1
    if errorlevel 1 (
      echo Error: resolved tag "!LATEST_TAG!" does not look like a version >&2
      exit /b 1
    )

    echo Falling back to !LATEST_TAG!... >&2
    set "VERSION=!LATEST_TAG!"
    set "ASSET=lumen-!VERSION:~1!-windows-!ARCH!.exe"
    set "URL=https://github.com/!REPO!/releases/download/!VERSION!/!ASSET!"

    call curl -sfL --max-time 300 --retry 3 --retry-delay 2 "!URL!" -o "!BINARY!"
    if errorlevel 1 (
      echo Error: fallback download also failed >&2
      exit /b 1
    )
  )

  echo Installed lumen to !BINARY! >&2
)

"%BINARY%" %*
