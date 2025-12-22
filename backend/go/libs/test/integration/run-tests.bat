@echo off
setlocal EnableExtensions EnableDelayedExpansion

rem Runs Go library test suites (integration, e2e) with optional docker-compose stack.
rem Usage: run-tests.bat [--suite integration^|e2e^|all] [--with-compose] [--keep-compose] [-- go test args]
rem Defaults to integration when no suite is provided.

set "SUITE=integration"
set "USE_COMPOSE=0"
set "KEEP_COMPOSE=0"
set "GO_ARGS="

:argloop
if "%~1"=="" goto args_done
if /I "%~1"=="--suite" goto handle_suite
if /I "%~1"=="--with-compose" (
  set "USE_COMPOSE=1"
  shift
  goto argloop
)
if /I "%~1"=="--keep-compose" (
  set "KEEP_COMPOSE=1"
  shift
  goto argloop
)
if "%~1"=="--" goto passthru_loop

set "GO_ARGS=%GO_ARGS% %~1"
shift
goto argloop

:handle_suite
if "%~2"=="" goto suite_error
set "SUITE=%~2"
shift
shift
goto argloop

:suite_error
echo [error] --suite requires a value (integration^|e2e^|all)
endlocal & exit /b 1

:passthru_loop
if "%~1"=="" goto args_done
set "GO_ARGS=%GO_ARGS% %~1"
shift
goto passthru_loop

:args_done
set "SCRIPT_DIR=%~dp0"
set ROOT_DIR=

pushd "%SCRIPT_DIR%" >nul
for /f "delims=" %%i in ('git rev-parse --show-toplevel 2^>nul') do set ROOT_DIR=%%i
if "%ROOT_DIR%"=="" (
  pushd ..\..\.. >nul
  set ROOT_DIR=%CD%
  popd >nul
)
popd >nul

if "%ROOT_DIR%"=="" (
  echo [error] could not locate repository root
  endlocal & exit /b 1
)

echo [info] suite=%SUITE% compose=%USE_COMPOSE% keep=%KEEP_COMPOSE%

pushd "%ROOT_DIR%" >nul
set "COMPOSE_FILE=%ROOT_DIR%\docker-compose.tests.yml"
set "STACK_STARTED=0"
set COMPOSE_CMD=
set TOPICS=auth.user.created tenant.created agent.created

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
  set "STACK_STARTED=1"

  echo [info] waiting for stack to be ready
  timeout /t 8 /nobreak >nul

  for %%T in (%TOPICS%) do (
    echo [info] ensuring topic %%T exists
    !COMPOSE_CMD! -f "%COMPOSE_FILE%" exec -T kafka kafka-topics --bootstrap-server localhost:9092 --create --if-not-exists --topic %%T --partitions 1 --replication-factor 1
  )
)

set LIBS=platform-core platform-auth platform-events platform-observability
for %%L in (%LIBS%) do (
  if /I "%SUITE%"=="integration" call :run_suite integration %%L
  if /I "%SUITE%"=="e2e" call :run_suite e2e %%L
  if /I "%SUITE%"=="all" (
    call :run_suite integration %%L
    call :run_suite e2e %%L
  )
)

if "%STACK_STARTED%"=="1" if not "%KEEP_COMPOSE%"=="1" (
  !COMPOSE_CMD! -f "%COMPOSE_FILE%" down
) else if "%STACK_STARTED%"=="1" (
  echo [info] leaving test stack running (requested with --keep-compose)
)

popd >nul
endlocal & exit /b 0

:run_suite
set "SUITE_NAME=%~1"
set "LIB_NAME=%~2"
set "LIB_DIR=%ROOT_DIR%\backend\go\libs\%LIB_NAME%"
set "TEST_DIR=!LIB_DIR!\test\%SUITE_NAME%"

if not exist "!TEST_DIR!" (
  echo [skip] %LIB_NAME% has no test\%SUITE_NAME%; skipping
  goto :eof
)

dir /b "!TEST_DIR!\*_test.go" >nul 2>&1
if errorlevel 1 (
  echo [skip] %LIB_NAME% test\%SUITE_NAME% has no *_test.go; skipping
  goto :eof
)

if /I "%LIB_NAME%"=="platform-events" if /I "%SUITE_NAME%"=="integration" (
  if "%KAFKA_BROKERS%"=="" set "KAFKA_BROKERS=localhost:9092"
)

echo [info] running %SUITE_NAME% tests for %LIB_NAME%
pushd "!LIB_DIR!" >nul
go test -tags=%SUITE_NAME% ./test/%SUITE_NAME% %GO_ARGS%
set "TEST_RESULT=!ERRORLEVEL!"
popd >nul
if not "%TEST_RESULT%"=="0" (
  echo [error] %LIB_NAME% %SUITE_NAME% tests failed with code %TEST_RESULT%
  if "%STACK_STARTED%"=="1" if not "%KEEP_COMPOSE%"=="1" !COMPOSE_CMD! -f "%COMPOSE_FILE%" down
  popd >nul
  endlocal & exit /b %TEST_RESULT%
)

echo [info] %LIB_NAME% %SUITE_NAME% tests passed
goto :eof

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
