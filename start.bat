@echo off
cd /d %~dp0
go run cmd/web/main.go
pause