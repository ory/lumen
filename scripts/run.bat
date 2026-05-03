@echo off
setlocal enabledelayedexpansion

:: Determine plugin root: prefer an agent-set env var, then fall back to the
:: repository layout so the same launcher works across supported hosts.
if defined CLAUDE_PLUGIN_ROOT (
  set "PLUGIN_ROOT=%CLAUDE_PLUGIN_ROOT%"
) else if defined CURSOR_PLUGIN_ROOT (
  set "PLUGIN_ROOT=%CURSOR_PLUGIN_ROOT%"
) else (
  set "PLUGIN_ROOT=%~dp0.."
)

:: Architecture detection
set "ARCH=amd64"
if "%PROCESSOR_ARCHITECTURE%"=="ARM64" set "ARCH=arm64"

:: Environment defaults
if not defined LUMEN_BACKEND set "LUMEN_BACKEND=ollama"
if not defined LUMEN_EMBED_MODEL set "LUMEN_EMBED_MODEL=ordis/jina-embeddings-v2-base-code"

:: Binary path
set "BINARY=%PLUGIN_ROOT%\bin\lumen-windows-%ARCH%.exe"

:: Download on first run if binary is missing
if not exist "%BINARY%" (
  set "REPO=ory/lumen"

  :: Always use the version pinned in the manifest — keeps plugin and binary in sync
  set "MANIFEST=%PLUGIN_ROOT%\.release-please-manifest.json"
  if not exist "!MANIFEST!" (
    echo Error: .release-please-manifest.json not found in %PLUGIN_ROOT% >&2
    exit /b 1
  )
  for /f "tokens=*" %%i in ('findstr /r "\"[.]\"" "!MANIFEST!"') do (
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
  if not exist "%PLUGIN_ROOT%\bin" mkdir "%PLUGIN_ROOT%\bin"

  call curl -sfL --max-time 300 --retry 3 --retry-delay 2 "!URL!" -o "%BINARY%"
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

    call curl -sfL --max-time 300 --retry 3 --retry-delay 2 "!URL!" -o "%BINARY%"
    if errorlevel 1 (
      echo Error: fallback download also failed >&2
      exit /b 1
    )
  )

  echo Installed lumen to %BINARY% >&2
)

:: Always ensure generic binary exists for PATH usage (backfills old installs)
set "GENERIC_BINARY=%PLUGIN_ROOT%\bin\lumen.exe"
if not exist "%GENERIC_BINARY%" (
  if exist "%BINARY%" (
    copy "%BINARY%" "%GENERIC_BINARY%" >nul
    if errorlevel 1 (
      echo Error: failed to create generic binary at %GENERIC_BINARY% >&2
      exit /b 1
    )
    echo Created %GENERIC_BINARY% for direct CLI usage >&2
  )
)

:: Auto-install to %USERPROFILE%\bin for CLI usage (no admin required).
:: Copy instead of symlink because mklink requires elevation on Windows.
if exist "%GENERIC_BINARY%" (
  set "USER_BIN=%USERPROFILE%\bin"
  set "USER_LUMEN=%USER_BIN%\lumen.exe"

  :: Create %USERPROFILE%\bin if it doesn't exist
  if not exist "!USER_BIN!" (
    mkdir "!USER_BIN!" >nul 2>&1
    if exist "!USER_BIN!" echo Created !USER_BIN! >&2
  )

  :: Copy to user bin if directory exists and file doesn't match
  if exist "!USER_BIN!" (
    set "NEEDS_COPY=0"
    if not exist "!USER_LUMEN!" set "NEEDS_COPY=1"
    if exist "!USER_LUMEN!" (
      fc /b "%GENERIC_BINARY%" "!USER_LUMEN!" >nul 2>&1
      if errorlevel 1 set "NEEDS_COPY=1"
    )

    if "!NEEDS_COPY!"=="1" (
      copy /y "%GENERIC_BINARY%" "!USER_LUMEN!" >nul 2>&1
      if exist "!USER_LUMEN!" (
        echo Installed lumen CLI to !USER_LUMEN! >&2

        :: Check if %USERPROFILE%\bin is in PATH
        echo %PATH% | findstr /i /c:"!USER_BIN!" >nul
        if errorlevel 1 (
          echo. >&2
          echo To use 'lumen' from anywhere, add to your PATH: >&2
          echo. >&2
          echo   1. Press Win+X, select "System" >&2
          echo   2. Click "Advanced system settings" ^> "Environment Variables" >&2
          echo   3. Under "User variables", edit "Path" >&2
          echo   4. Add: %USERPROFILE%\bin >&2
          echo. >&2
        )
      )
    )
  )
)

"%BINARY%" %*
