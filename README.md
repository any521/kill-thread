# kill-port —— Windows 端口查询 / 进程终止工具

一个免安装、免依赖的 Windows 小工具：**查询哪个程序占用了端口**、**一键结束占用端口的进程**。

v1.1.0 起提供两种形态：

| 文件 | 形态 | 适合 |
|---|---|---|
| **kill-port.exe** | 🖥️ **图形界面桌面应用**（无黑框，双击直接用） | 日常使用 |
| **kill-port-cli.exe** | ⌨️ 命令行 / 交互菜单（控制台） | 脚本、远程、服务器 |

## 下载（点击即可）

* **⭐ [下载 kill-port.exe（图形界面版 v1.1.0）](https://github.com/any521/kill-thread/releases/download/v1.1.0/kill-port.exe)**
* [下载 kill-port-cli.exe（命令行版 v1.1.0）](https://github.com/any521/kill-thread/releases/download/v1.1.0/kill-port-cli.exe)
* 版本总览：[Latest Release](https://github.com/any521/kill-thread/releases/latest)

均为 Windows x64 单文件（约 2.3 MB），免安装、免 .NET/VC++ 运行库，下载后**双击即用**。

## 图形界面版使用

双击 `kill-port.exe` 直接打开窗口：

```text
┌──────────────────────────────────────────────────────────────┐
│ kill-port —— 端口查询 / 进程终止  v1.1.0                      │
├──────────────────────────────────────────────────────────────┤
│ [刷新] [结束选中进程] [结束该端口全部进程] ☑自动刷新  [过滤🔍] │
│ ┌────────┬──────────────────┬───────┬────────┬────────────┐  │
│ │ 协议   │ 本地地址          │ 端口  │ PID    │ 进程名     │  │
│ ├────────┼──────────────────┼───────┼────────┼────────────┤  │
│ │ TCP    │ 0.0.0.0:8080     │ 8080  │ 16384  │ java.exe   │  │
│ │ TCP    │ 127.0.0.1:3000   │ 3000  │ 4021   │ node.exe   │  │
│ │ UDP    │ 0.0.0.0:5353     │ 5353  │ 1080   │ svchost.ex │  │
│ └────────┴──────────────────┴───────┴────────┴────────────┘  │
│ 显示 24 / 共 24 条   刷新于 15:04:05   （当前非管理员…）       │
└──────────────────────────────────────────────────────────────┘
```

* **列表**：协议 / 本地地址 / 端口 / PID / 进程名，多选、点表头排序
* **过滤框**：输入端口号 / 进程名 / 协议实时筛选（如输 `8080` 或 `java`）
* **结束选中进程**：勾选一行或多行 → 点击按钮 → 确认弹窗 → 结束
* **结束该端口全部进程**：结束选中行所在端口的全部占用进程（TCP+UDP、IPv4+IPv6 一次清空）
* **双击某行** = 直接结束该行进程；**自动刷新** 每 2 秒同步一次端口状态
* 结束前都有二次确认弹窗；结果（成功/失败清单）用对话框反馈
* 窗口底部提示当前是否具备管理员权限

图形界面版用 **纯 Win32 API（syscall）实现**，无任何 WebView/运行库依赖，所以体积仍然只有 2.3 MB、启动秒开。

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

深色圆角方块 + 红色电源/终止符号 + 绿色终端角标，已嵌入 exe 文件本身（文件属性页也显示图标与版本信息 v1.1.0）：

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

## 常见问题

* **双击没反应？** 首次运行若被 SmartScreen 拦截，点“更多信息 → 仍要运行”（个人工具未做代码签名所致，源码开源可自行编译验证）。
* **结束失败？** 窗口底部显示“当前非管理员”时，请右键以管理员身份运行。
* **杀软告警？** 本工具会调用 taskkill 结束进程，个别杀软可能提示，添加信任即可。

## License

MIT
