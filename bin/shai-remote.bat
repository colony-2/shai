@echo off
setlocal

rem Get the directory where this batch file is located
set "SCRIPT_DIR=%~dp0"

rem Path to the actual shell script
set "TARGET=%SCRIPT_DIR%..\internal\shai\runtime\bootstrap\shai-remote.sh"

rem Execute the shell script using bash (available in Git Bash on Windows)
bash "%TARGET%" %*
