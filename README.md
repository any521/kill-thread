# kill-port —— Windows 端口查询 / 进程终止工具（单文件 exe）

一个免安装、免依赖的命令行小工具：**查询哪个程序占用了端口**、**一键结束占用端口的进程**。

直接双击 `kill-port.exe` 即进入中文交互菜单；也支持命令行参数快速操作。

## 下载（点击即可运行）

- **⭐ [点击直接下载 kill-port.exe（Release 直链）](https://github.com/any521/kill-tread/releases/download/v1.0.0/kill-port.exe)** —— Windows x64，约 2.3 MB，免安装绿色单文件，下载后双击即可使用
- 版本总览：[Latest Release](https://github.com/any521/kill-tread/releases/latest)
- 源码仓库里也保留了一份 [kill-port.exe](./kill-port.exe)

程序图标（深色圆角方块 + 红色电源/终止符号 + 绿色终端角标），已嵌入 exe 文件本身：

![icon](./icon_preview.png)
2. 双击运行进入菜单，或在命令行中使用：

```text
kill-port.exe                 进入交互菜单
kill-port.exe list            列出所有监听端口及占用进程
kill-port.exe query 8080      查询 8080 端口被哪个程序占用
kill-port.exe kill 8080       结束占用 8080 端口的程序（会先确认）
kill-port.exe kill 8080 -y    结束时跳过确认（适合脚本）
kill-port.exe killpid 1234    按 PID 强制结束进程
kill-port.exe help            查看帮助
```

> 提示：结束系统进程或其它用户的进程时，请**以管理员身份**运行。

## 交互菜单示例

```text
==============================================
        kill-port  端口查询 / 进程终止
==============================================
──────────────────────────────────────────────
  1. 列出所有端口及占用程序
  2. 查询指定端口被哪个程序占用
  3. 关闭占用指定端口的程序
  4. 按 PID 强制结束进程
  0. 退出
请选择> 3
请输入端口号 (1-65535)> 8080

协议   本地地址                    端口     PID      进程名
────────────────────────────────────────────────────────────────────────
TCP    0.0.0.0:8080               8080     16384    java.exe
确定要结束占用端口 8080 的 1 个进程吗? [y/N]> y
✔ 已结束 java.exe (PID=16384)
端口 8080 已释放。
```

## 功能特点

- **单文件 exe**：Go 静态编译，无需安装 .NET / VC++ 运行库，拷到即用。
- **中文界面**：自动把控制台切换为 UTF-8 代码页，中文不乱码。
- **查询**：解析 `netstat -ano` + `tasklist`，显示协议 / 地址 / 端口 / PID / 进程名。
- **终止**：`taskkill /F /PID` 强制结束，支持一个端口的多个进程去重处理，结束前默认二次确认。
- **跨平台**：同一份源码也支持 Linux / macOS（`netstat -tulnp` / `ss -ltnp` + SIGKILL），仓库附带 Linux 版编译命令。

## 从源码构建

安装 [Go](https://go.dev/dl/) 1.22+ 后：

```bash
# Windows exe
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o kill-port.exe .
# Linux 二进制
go build -o kill-port .
```

Windows 用户可直接执行 `build.bat`（需已安装 Go）。

## 常见问题

- **双击闪退？** 请在命令行/cmd 中运行，或使用交互菜单方式；无参数运行不会闪退。
- **结束失败？** 右键"以管理员身份运行"。
- **误报杀毒？** 工具会调用 taskkill 结束进程，个别杀软可能告警，添加信任即可（源码开放可自行编译验证）。

## License

MIT
