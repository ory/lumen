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
set "TMP_BINARY=%BINARY%.tmp"

:: Download on first run if binary is missing
if not exist "%BINARY%" (
  if defined LUMEN_RELEASE_REPO (
    set "REPO=%LUMEN_RELEASE_REPO%"
    powershell -NoProfile -Command "if ($env:REPO -match '^[A-Za-z0-9][A-Za-z0-9-]*/[A-Za-z0-9_.-][A-Za-z0-9_.-]*$' -and $env:REPO -notmatch '\.git$') { exit 0 } else { exit 1 }" >nul 2>&1
    if errorlevel 1 (
      echo Error: LUMEN_RELEASE_REPO must be in owner/repo form >&2
      exit /b 1
    )
    for /f "tokens=1 delims=/" %%o in ("!REPO!") do set "REPO_OWNER=%%o"
    if "!REPO_OWNER:~-1!"=="-" (
      echo Error: LUMEN_RELEASE_REPO must be in owner/repo form >&2
      exit /b 1
    )
    if not "!REPO_OWNER:~39,1!"=="" (
      echo Error: LUMEN_RELEASE_REPO must be in owner/repo form >&2
      exit /b 1
    )
    if not "!REPO_OWNER:--=!"=="!REPO_OWNER!" (
      echo Error: LUMEN_RELEASE_REPO must be in owner/repo form >&2
      exit /b 1
    )
  ) else (
    set "REPO="
    set "ORIGIN_URL="
    for /f "delims=" %%u in ('git -C "%PLUGIN_ROOT%" remote get-url origin 2^>nul') do (
      if not defined ORIGIN_URL set "ORIGIN_URL=%%u"
    )
    if defined ORIGIN_URL (
      set "CANDIDATE="
      set "_TMP=!ORIGIN_URL!"
      if /I "!_TMP:~0,19!"=="https://github.com/" (
        set "CANDIDATE=!_TMP:~19!"
      ) else if /I "!_TMP:~0,15!"=="git@github.com:" (
        set "CANDIDATE=!_TMP:~15!"
      ) else if /I "!_TMP:~0,21!"=="ssh://git@github.com/" (
        set "CANDIDATE=!_TMP:~21!"
      )
      if defined CANDIDATE (
        if "!CANDIDATE:~-4!"==".git" set "CANDIDATE=!CANDIDATE:~0,-4!"
        set "CANDIDATE_OK=0"
        powershell -NoProfile -Command "if ($env:CANDIDATE -match '^[A-Za-z0-9][A-Za-z0-9-]*/[A-Za-z0-9_.-][A-Za-z0-9_.-]*$' -and $env:CANDIDATE -notmatch '\.git$') { exit 0 } else { exit 1 }" >nul 2>&1
        if not errorlevel 1 set "CANDIDATE_OK=1"
        if "!CANDIDATE_OK!"=="1" (
          for /f "tokens=1 delims=/" %%o in ("!CANDIDATE!") do set "CANDIDATE_OWNER=%%o"
          if "!CANDIDATE_OWNER:~-1!"=="-" set "CANDIDATE_OK=0"
          if not "!CANDIDATE_OWNER:~39,1!"=="" set "CANDIDATE_OK=0"
          if not "!CANDIDATE_OWNER:--=!"=="!CANDIDATE_OWNER!" set "CANDIDATE_OK=0"
        )
        if "!CANDIDATE_OK!"=="1" set "REPO=!CANDIDATE!"
      )
    )
    if not defined REPO set "REPO=ory/lumen"
  )

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

  if exist "%TMP_BINARY%" del "%TMP_BINARY%" 2>nul
  call curl -sfL --max-time 300 --retry 3 --retry-delay 2 "!URL!" -o "%TMP_BINARY%"
  if errorlevel 1 (
    if exist "%TMP_BINARY%" del "%TMP_BINARY%" 2>nul
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

    if exist "%TMP_BINARY%" del "%TMP_BINARY%" 2>nul
    call curl -sfL --max-time 300 --retry 3 --retry-delay 2 "!URL!" -o "%TMP_BINARY%"
    if errorlevel 1 (
      if exist "%TMP_BINARY%" del "%TMP_BINARY%" 2>nul
      echo Error: fallback download also failed >&2
      exit /b 1
    )
  )

  move /Y "%TMP_BINARY%" "%BINARY%" >nul
  if errorlevel 1 (
    if exist "%TMP_BINARY%" del "%TMP_BINARY%" 2>nul
    echo Error: could not install downloaded lumen binary >&2
    exit /b 1
  )
  echo Installed lumen to %BINARY% >&2
)

set "LUMEN_PLUGIN_ROOT=%PLUGIN_ROOT%"
"%BINARY%" %*
