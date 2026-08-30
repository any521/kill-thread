# kill-port —— Windows 端口查询 / 进程终止工具

一个免安装、免依赖的 Windows 小工具：**查询哪个程序占用了端口**、**一键结束占用端口的进程**。

v1.1.0 起提供两种形态：

| 文件 | 形态 | 适合 |
|---|---|---|
| **kill-port.exe** | 🖥️ **图形界面桌面应用**（无黑框，双击直接用） | 日常使用 |
| **kill-port-cli.exe** | ⌨️ 命令行 / 交互菜单（控制台） | 脚本、远程、服务器 |

## 下载（点击即可）

* **⭐ [下载 kill-port.exe（图形界面版 v1.1.2）](https://github.com/any521/kill-thread/releases/download/v1.4.0/kill-port.exe)**
* [下载 kill-port-cli.exe（命令行版 v1.1.2）](https://github.com/any521/kill-thread/releases/download/v1.4.0/kill-port-cli.exe)
* 版本总览：[Latest Release](https://github.com/any521/kill-thread/releases/latest)

均为 Windows x64 单文件（约 2.3 MB），免安装、免 .NET/VC++ 运行库，下载后**双击即用**。

## 图形界面版（微信风格端口管理）

双击 `kill-port.exe` 直接打开管理页面（无控制台黑框）：

```text
◤ ┃ 端口管理                                ( 🔍 搜索端口 / 进程名 )
◢ ┃ 24 个监听端口 · 更新于 15:04:05        [ 刷新 ] [ 结束该端口全部进程 ]   自动刷新 (●─)
▓ ┃ ┌──────────────────────────────────────────────────────────────┐
▓ ┃ │ (TCP) 8080   java.exe                        ( 结束进程 )    │
▓ ┃ │      PID 16384 · 0.0.0.0:8080                                │
▓ ┃ ├──────────────────────────────────────────────────────────────┤
▓ ┃ │ (UDP) 5353   svchost.exe                     ( 结束进程 )    │
▓ ┃ │      PID 1080 · 0.0.0.0:5353                                 │
▓ ┃ └──────────────────────────────────────────────────────────────┘
▓ ┃ 显示 24 / 24 条                                管理员权限已就绪
```

* **搜索**：右上角搜索框实时过滤，输入端口号（`8080`）、进程名（`java`）、协议（`tcp`）均可，`Esc` 清空
* **结束单个进程**：点行内红色「结束进程」按钮或双击该行 → 确认弹窗 → 结束
* **结束端口全部进程**：先点选一行（绿色高亮），再点顶部绿色按钮，一次清空该端口所有占用（TCP/UDP、IPv4/IPv6）
* **自动刷新**：微信绿滑动开关，默认每 2 秒后台同步（刷新/结束进程均不卡 UI）
* 悬停高亮、行选中、细滚动条、滚轮翻页、管理员状态提示一应俱全
* **一键 UAC 提权补杀**：结束 mysqld.exe / svchost.exe 等系统进程遇到 Access denied 时，自动询问「立即以管理员权限补杀」→ 弹一次 UAC 确认即静默完成，无需重启程序；右下角橙色提示也可点击提权重启
* **左侧菜单**：「端口」= 管理主页；「日志」= 运行日志页（启动记录、结束进程操作、异常信息，微信绿高亮当前页，支持滚轮翻阅、一键用记事本打开日志文件）

> **提示**：mysqld 等 Windows 服务进程被结束后，服务控制管理器可能自动重启它。想彻底停止请结束前在命令行执行 `net stop mysql`（服务名可用 `sc queryex type=service state=all | findstr /i mysql` 查询），或在「服务」管理器中禁用。
>
> 搜索框支持英文与数字输入（端口号/进程名检索足够）；如需粘贴中文进程名，可用命令行版 `query` 命令。

## 命令行版使用

```text
kill-port-cli.exe                 进入中文交互菜单
kill-port-cli.exe list            列出所有监听端口及占用进程
kill-port-cli.exe query 8080      查询 8080 端口被哪个程序占用
kill-port-cli.exe kill 8080       结束占用 8080 端口的程序（先确认）
kill-port-cli.exe kill 8080 -y    跳过确认（脚本用）
kill-port-cli.exe killpid 1234    按 PID 强制结束进程
kill-port-cli.exe help            查看帮助
```

> 提示：结束系统进程或其它用户的进程时，请**右键 → 以管理员身份运行**。

## 程序图标

深色圆角方块 + 红色电源/终止符号 + 绿色终端角标，已嵌入 exe 文件本身（文件属性页也显示图标与版本信息 v1.1.2）：

![icon](./icon_preview.png)

图标由 `tools/genicon.go` 纯代码绘制（SDF 抗锯齿，7 尺寸 ICO），可一键重新生成。

## 从源码构建

安装 [Go](https://go.dev/dl/) 1.22+ 后（交叉编译不需要 Windows / C 编译器）：

```bash
# 图形界面版（无控制台黑框）
GOOS=windows GOARCH=amd64 go build -trimpath -tags gui -ldflags "-s -w -H windowsgui" -o kill-port.exe .
# 命令行版
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o kill-port-cli.exe .
# Linux 版（控制台）
go build -o kill-port .
```

Windows 用户直接运行 `build.bat`；修改图标后先 `go run tools/genicon.go` 再 `go-winres make`。

## 双击 exe 没反应？按顺序排查

v1.1.1 起程序自带**启动日志**：任何启动失败都会弹窗提示，并记录到 `%TEMP%` 目录下的 `kill-port.log`（Win+R 输入 `%TEMP%` 回车即可找到）。排查步骤：

1. **任务管理器**里找 `kill-port.exe`：若在进程列表里但没有窗口 → 查看日志文件末尾的错误信息。
2. **杀软静默拦截（国内最常见）**：360/电脑管家/火绒会拦截调用 taskkill 的小工具且**不弹任何窗**。先完全退出杀软再双击；成功后把 exe 加入信任区，或到隔离区恢复文件。
3. **SmartScreen**：蓝底提示点“更多信息 → 仍要运行”。
4. **文件名/锁定**：浏览器下载可能改名或带“来自其他计算机”锁定标记——右键 → 属性 → 勾选“**解除锁定**”→ 应用。
5. **系统版本**：v1.1.1 已改用支持 **Windows 7 SP1 及以上** 的 Go 编译；老 Win7 若提示缺补丁，先装系统更新或换 Win10/11。
6. 仍不行：在 cmd 里运行 `kill-port-cli.exe list`，命令行版能看到具体报错。

## 常见问题

* **双击没反应？** 见上面《双击 exe 没反应？》专节。
* **刷新/结束进程闪黑框？** 已修复：v1.1.1 起子进程（netstat/tasklist/taskkill）全部隐藏窗口运行。
* **结束失败？** 窗口底部显示“当前非管理员”时，右键以管理员身份运行。
* **误报杀毒？** 本工具调用 taskkill 结束进程，个别杀软会告警，添加信任即可（源码开源可自行编译验证）。

## License

MIT