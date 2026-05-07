@echo off
setlocal enabledelayedexpansion

:: Test that run.cmd correctly delegates to run.bat on Windows.
:: Mirrors test_run_cmd.sh but for the batch (Windows) code path.

set "PASS=0"
set "FAIL=0"

set "TMP_DIR=%TEMP%\lumen-test-%RANDOM%"
mkdir "%TMP_DIR%" 2>NUL

:: Copy run.cmd to temp dir
copy "%~dp0run.cmd" "%TMP_DIR%\run.cmd" >NUL

:: Create a mock run.bat that echoes its arguments
(
  echo @echo off
  echo echo delegated:%%*
) > "%TMP_DIR%\run.bat"

:: --- Test 1: run.cmd delegates to run.bat with correct arguments ---
set "OUTPUT="
for /f "tokens=*" %%i in ('"%TMP_DIR%\run.cmd" stdio --flag 2^>NUL') do set "OUTPUT=%%i"

if "!OUTPUT!"=="delegated:stdio --flag" (
  echo   PASS: run.cmd delegates to run.bat with correct arguments
  set /a PASS+=1
) else (
  echo   FAIL: run.cmd delegates to run.bat with correct arguments
  echo         expected: delegated:stdio --flag
  echo         got:      !OUTPUT!
  set /a FAIL+=1
)

:: --- Test 2: run.cmd delegates hook subcommand ---
set "OUTPUT="
for /f "tokens=*" %%i in ('"%TMP_DIR%\run.cmd" hook session-start lumen --host claude 2^>NUL') do set "OUTPUT=%%i"

if "!OUTPUT!"=="delegated:hook session-start lumen --host claude" (
  echo   PASS: run.cmd delegates hook subcommand
  set /a PASS+=1
) else (
  echo   FAIL: run.cmd delegates hook subcommand
  echo         expected: delegated:hook session-start lumen --host claude
  echo         got:      !OUTPUT!
  set /a FAIL+=1
)

:: --- Test 3: run.cmd produces no unexpected stderr ---
set "STDERR_FILE=%TMP_DIR%\stderr.txt"
"%TMP_DIR%\run.cmd" stdio 2>"%STDERR_FILE%" >NUL

:: Check stderr is empty or contains only the expected "not recognized" from shebang line
set "STDERR_SIZE=0"
for %%A in ("%STDERR_FILE%") do set "STDERR_SIZE=%%~zA"

:: Stderr should only contain the harmless shebang error (if any)
:: We check it doesn't contain "-S" which was the old broken error
findstr /i "\-S" "%STDERR_FILE%" >NUL 2>&1
if errorlevel 1 (
  echo   PASS: no '-S' error in stderr
  set /a PASS+=1
) else (
  echo   FAIL: stderr contains '-S' error from old broken polyglot
  type "%STDERR_FILE%"
  set /a FAIL+=1
)

:: Cleanup
rmdir /s /q "%TMP_DIR%" 2>NUL

:: Summary
echo.
echo === summary ===
echo   passed: %PASS%
echo   failed: %FAIL%

if %FAIL% GTR 0 exit /b 1
exit /b 0
