@echo off
setlocal enabledelayedexpansion

rem Runs tenant-manager integration tests with optional docker-compose stack.
rem Usage: run-integration-tests.bat [--with-compose] [--keep-compose] [-- go test args]

set USE_COMPOSE=0
set KEEP_COMPOSE=0
set PASSTHRU=0
set GO_ARGS=

:parse_args
if "%~1"=="" goto after_parse
if %PASSTHRU%==1 (
  set GO_ARGS=%GO_ARGS% %1
  shift
  goto parse_args
)
if "%~1"=="--with-compose" (
  set USE_COMPOSE=1
  shift
  goto parse_args
)
if "%~1"=="--keep-compose" (
  set KEEP_COMPOSE=1
  shift
  goto parse_args
)
if "%~1"=="--" (
  set PASSTHRU=1
  shift
  goto parse_args
)
if "%~1"=="--help" goto show_help
if "%~1"=="-h" goto show_help
set GO_ARGS=%GO_ARGS% %1
shift
goto parse_args

:show_help
echo Usage: %~n0 [--with-compose] [--keep-compose] [-- go test args]
endlocal & exit /b 0

:after_parse
set SCRIPT_DIR=%~dp0
set ROOT_DIR=

pushd "%SCRIPT_DIR%" >nul
for /f "delims=" %%i in ('git rev-parse --show-toplevel 2^>nul') do set ROOT_DIR=%%i
if "%ROOT_DIR%"=="" (
  pushd ..\..\..\.. >nul
  set ROOT_DIR=%CD%
  popd >nul
)
popd >nul

if "%ROOT_DIR%"=="" (
  echo [error] could not locate repository root
  endlocal & exit /b 1
)

pushd "%ROOT_DIR%" >nul
set COMPOSE_FILE=%ROOT_DIR%\docker-compose.tests.yml
set STACK_STARTED=0
set COMPOSE_CMD=

if "%USE_COMPOSE%"=="1" (
  call :detect_compose
  if "!COMPOSE_CMD!"=="" (
    echo [error] docker compose is required for --with-compose
    popd >nul
    endlocal & exit /b 1
  )

  echo [info] starting test stack using %COMPOSE_FILE%
  !COMPOSE_CMD! -f "%COMPOSE_FILE%" up -d
  if errorlevel 1 (
    echo [error] failed to start docker compose stack
    popd >nul
    endlocal & exit /b 1
  )
  set STACK_STARTED=1

  echo [info] waiting for stack to be ready
  timeout /t 8 /nobreak >nul
)

echo [info] running tenant-manager integration tests
pushd "%ROOT_DIR%\backend\go\services\tenant-manager" >nul
go test -tags=integration ./test/integration %GO_ARGS%
set TEST_RESULT=!ERRORLEVEL!
popd >nul

if not "%TEST_RESULT%"=="0" (
  if "%STACK_STARTED%"=="1" if not "%KEEP_COMPOSE%"=="1" %COMPOSE_CMD% -f "%COMPOSE_FILE%" down
  popd >nul
  endlocal & exit /b %TEST_RESULT%
)

if "%STACK_STARTED%"=="1" if not "%KEEP_COMPOSE%"=="1" (
  %COMPOSE_CMD% -f "%COMPOSE_FILE%" down
) else if "%STACK_STARTED%"=="1" (
  echo [info] leaving test stack running (requested with --keep-compose)
)

popd >nul
endlocal & exit /b 0

:detect_compose
set COMPOSE_CMD=
docker-compose version >nul 2>&1
if not errorlevel 1 (
  set COMPOSE_CMD=docker-compose
  goto :eof
)
docker compose version >nul 2>&1
if not errorlevel 1 (
  set COMPOSE_CMD=docker compose
)
goto :eof
