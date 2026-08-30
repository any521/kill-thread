@echo off
rem 需要已安装 Go 1.22+ ：https://go.dev/dl/
go build -trimpath -ldflags "-s -w" -o kill-port.exe .
if %errorlevel%==0 (echo 编译成功: kill-port.exe) else (echo 编译失败，请确认已安装 Go)
pause
