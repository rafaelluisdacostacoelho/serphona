@echo off
setlocal enabledelayedexpansion

set USE_COMPOSE=0
set KEEP_COMPOSE=0
set GO_ARGS=

:parse_args
if "%~1"=="" goto after_parse
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
if "%~1"=="--help" goto show_help
if "%~1"=="-h" goto show_help
set GO_ARGS=%GO_ARGS% %1
shift
goto parse_args

:show_help
echo Usage: %~n0 [--with-compose] [--keep-compose] [go test args]
endlocal & exit /b 0

:after_parse
set SCRIPT_DIR=%~dp0
set ROOT_DIR=

pushd "%SCRIPT_DIR%" >nul
for /f "delims=" %%i in ('git rev-parse --show-toplevel 2^>nul') do set ROOT_DIR=%%i
if "%ROOT_DIR%"=="" (
  pushd ..\..\..\..\..\.. >nul
  set ROOT_DIR=%CD%
  popd >nul
)
popd >nul

if "%ROOT_DIR%"=="" (
  echo [error] could not locate repository root
  endlocal & exit /b 1
)

pushd "%ROOT_DIR%" >nul
set ROOT_DIR=%CD%

if "%KAFKA_BROKERS%"=="" set KAFKA_BROKERS=localhost:9092
echo [info] using KAFKA_BROKERS=%KAFKA_BROKERS%

set COMPOSE_FILE=%ROOT_DIR%\docker-compose.tests.yml
set STACK_STARTED=0
set MODULE_DIR=%ROOT_DIR%\backend\go\libs\platform-events
set GO_TEST_PATH=./test/integration

if "%USE_COMPOSE%"=="1" (
  call :detect_compose
  if "!COMPOSE_CMD!"=="" (
    echo [error] docker compose is required for --with-compose
    popd >nul
    endlocal & exit /b 1
  )

  echo [info] starting Kafka test stack using %COMPOSE_FILE%
  !COMPOSE_CMD! -f "%COMPOSE_FILE%" up -d
  if errorlevel 1 (
    echo [error] failed to start docker compose stack
    popd >nul
    endlocal & exit /b 1
  )
  set STACK_STARTED=1

  echo [info] waiting for Kafka to be ready
  timeout /t 8 /nobreak >nul

  for %%T in (auth.user.created tenant.created agent.created) do (
    echo [info] ensuring topic %%T exists
    !COMPOSE_CMD! -f "%COMPOSE_FILE%" exec -T kafka kafka-topics --bootstrap-server localhost:9092 --create --if-not-exists --topic %%T --partitions 1 --replication-factor 1
    if errorlevel 1 (
      echo [error] failed to create topic %%T
      if not "%KEEP_COMPOSE%"=="1" !COMPOSE_CMD! -f "%COMPOSE_FILE%" down
      popd >nul
      endlocal & exit /b 1
    )
    call :wait_topic %%T
  )

  echo [info] waiting for topic metadata to propagate
  timeout /t 3 /nobreak >nul
)

echo [info] running integration tests in %MODULE_DIR%/%GO_TEST_PATH%
pushd "%MODULE_DIR%" >nul
go test -tags=integration %GO_TEST_PATH% %GO_ARGS%
set TEST_RESULT=%ERRORLEVEL%
popd >nul

if "%STACK_STARTED%"=="1" if not "%KEEP_COMPOSE%"=="1" (
  %COMPOSE_CMD% -f "%COMPOSE_FILE%" down
) else if "%STACK_STARTED%"=="1" (
  echo [info] leaving Kafka test stack running (requested with --keep-compose)
)

popd >nul
endlocal & exit /b %TEST_RESULT%

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

:wait_topic
set TOPIC_NAME=%1
set TOPIC_READY=0
for /l %%i in (1,1,10) do (
  !COMPOSE_CMD! -f "%COMPOSE_FILE%" exec -T kafka kafka-topics --bootstrap-server localhost:9092 --describe --topic %TOPIC_NAME% >nul 2>&1
  if not errorlevel 1 (
    set TOPIC_READY=1
    goto :wait_topic_done
  )
  timeout /t 1 /nobreak >nul
)
:wait_topic_done
if "%TOPIC_READY%"=="0" (
  echo [error] topic %TOPIC_NAME% did not become available
  if not "%KEEP_COMPOSE%"=="1" !COMPOSE_CMD! -f "%COMPOSE_FILE%" down
  popd >nul
  endlocal & exit /b 1
)
goto :eof
