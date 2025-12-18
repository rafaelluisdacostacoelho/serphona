@echo off
setlocal enabledelayedexpansion

rem Backward-compatible shim that delegates to the shared test\integration runner for integration suite only.
set SCRIPT_DIR=%~dp0
call "%SCRIPT_DIR%test\integration\run-tests.bat" --suite integration %*
endlocal
