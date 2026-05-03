:<<"::CMDLITERAL"
@echo off
goto :batch
::CMDLITERAL
exec "$(dirname "$0")/run.sh" "$@"

:batch
@echo off
set "SCRIPT_DIR=%~dp0"
call "%SCRIPT_DIR%run.bat" %*
exit /b %ERRORLEVEL%
