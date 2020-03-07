rem @echo off

set GOARCH=amd64

set GOOS=linux

set TARGETEXT=

set OUTPUTDIR=bin\

mkdir %OUTPUTDIR%

set GOHOSTARCH=amd64

set GOHOSTOS=windows

set TARGETDIR=%OUTPUTDIR%\%GOOS%-%GOARCH%

mkdir %TARGETDIR%

for %%i in ("%~dp0\.") do (
  set CurDir=%%~ni
)

set TARGET=bin\%CurDir%-%GOOS%-%GOARCH%%TARGETEXT%

set cnf=%CurDir%.example

echo building %TARGET% ...

go build  -o %TARGET%

copy %TARGET% %TARGETDIR%
copy %cnf% %TARGETDIR%

echo "OK"

pause