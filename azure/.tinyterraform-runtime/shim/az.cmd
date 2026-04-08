@echo off
pwsh -NoProfile -File "%~dp0azshim.ps1" %*
exit /b %ERRORLEVEL%
