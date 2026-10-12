@echo off
setlocal
chcp 65001 >nul
cd /d "%~dp0"
"%SystemRoot%\System32\where.exe" git >nul 2>nul
if errorlevel 1 (
  echo Install Git for Windows, then reopen Gitra.cmd.
  echo https://gitforwindows.org/
  pause
  exit /b 1
)
"%~dp0gitra.exe" %*
set "gitra_exit=%errorlevel%"
if not "%gitra_exit%"=="0" pause
exit /b %gitra_exit%
