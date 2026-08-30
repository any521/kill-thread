@echo off
rem 需要已安装 Go 1.22+ ：https://go.dev/dl/
echo ==^> kill-port.exe (GUI 桌面版, 无黑框)
go build -trimpath -tags gui -ldflags "-s -w -H windowsgui" -o kill-port.exe .
echo ==^> kill-port-cli.exe (命令行版)
go build -trimpath -ldflags "-s -w" -o kill-port-cli.exe .
if %errorlevel%==0 (echo 编译成功) else (echo 编译失败，请确认已安装 Go)
pause
