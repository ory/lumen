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

:: --- Test 3: run.cmd produces clean stderr ---
:: The polyglot prologue must not surface command-not-found errors
:: from cmd.exe trying to execute the shell-only lines.
:: NOTE: invoke via `cmd /c` to match how MCP hosts spawn run.cmd
:: (a fresh cmd.exe with default echo state, not inheriting `call`).
set "STDERR_FILE=%TMP_DIR%\stderr.txt"
set "STDOUT_FILE=%TMP_DIR%\stdout.txt"
cmd /c ""%TMP_DIR%\run.cmd" stdio" 2>"%STDERR_FILE%" >"%STDOUT_FILE%"

set "STDERR_SIZE=0"
for %%A in ("%STDERR_FILE%") do set "STDERR_SIZE=%%~zA"

if "!STDERR_SIZE!"=="0" (
  echo   PASS: stderr is empty
  set /a PASS+=1
) else (
  echo   FAIL: stderr is not empty [size: !STDERR_SIZE! bytes]
  type "%STDERR_FILE%"
  set /a FAIL+=1
)

:: --- Test 4: run.cmd produces clean stdout (no command-echo pollution) ---
:: cmd.exe echoes commands to stdout before @echo off takes effect, so a
:: polyglot whose first line is treated as a command (rather than a label
:: or @-prefixed line) prepends prompt-prefixed text to stdout. This breaks
:: MCP stdio JSON-RPC framing on Claude Code, Cursor, and other hosts that
:: spawn run.cmd directly via .cmd file extension association.
set "STDOUT_FIRST="
for /f "usebackq delims=" %%i in ("%STDOUT_FILE%") do (
  if not defined STDOUT_FIRST set "STDOUT_FIRST=%%i"
)
for /f %%i in ('find /c /v "" ^< "%STDOUT_FILE%"') do set "STDOUT_LINES=%%i"

set "TEST4=fail"
if "!STDOUT_FIRST!"=="delegated:stdio" if "!STDOUT_LINES!"=="1" set "TEST4=pass"

if "!TEST4!"=="pass" (
  echo   PASS: stdout has no command-echo pollution
  set /a PASS+=1
) else (
  echo   FAIL: stdout polluted before run.bat output
  echo         expected: 1 line "delegated:stdio"
  echo         got first line: !STDOUT_FIRST!
  echo         line count:     !STDOUT_LINES!
  echo         --- full stdout ---
  type "%STDOUT_FILE%"
  echo         --- end ---
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
