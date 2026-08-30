//go:build !gui

// kill-port 命令行版入口（控制台）。GUI 版见 gui_windows.go（-tags gui 编译）。
package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

const helpText = `kill-port —— 端口查询 / 端口进程终止工具 (命令行版)

用法:
  kill-port-cli                 进入交互菜单
  kill-port-cli list            列出所有监听端口及占用进程
  kill-port-cli query <端口>    查询某端口被哪个进程占用
  kill-port-cli kill <端口>     结束占用某端口的进程   (加 -y 跳过确认)
  kill-port-cli killpid <PID>   按 PID 强制结束进程     (加 -y 跳过确认)
  kill-port-cli help            显示本帮助

提示: 结束系统进程或他人进程时，Windows 请以管理员身份运行。`

func main() {
	setupConsole()

	args := os.Args[1:]
	if len(args) == 0 {
		interactiveMenu()
		return
	}

	switch strings.ToLower(args[0]) {
	case "list", "ls":
		if err := doList(); err != nil {
			fail(err)
		}
	case "query", "q", "find", "who":
		if len(args) < 2 {
			fail(fmt.Errorf("缺少端口号，例如: kill-port-cli query 8080"))
		}
		port, err := parsePort(args[1])
		if err != nil {
			fail(err)
		}
		conns, err := findPort(port)
		if err != nil {
			fail(err)
		}
		if len(conns) == 0 {
			fmt.Printf("端口 %d 当前没有被任何程序占用。\n", port)
			return
		}
		printTable(conns)
	case "kill", "k":
		if len(args) < 2 {
			fail(fmt.Errorf("缺少端口号，例如: kill-port-cli kill 8080"))
		}
		port, err := parsePort(args[1])
		if err != nil {
			fail(err)
		}
		if err := killPort(port, hasFlag(args[2:], "-y", "--yes")); err != nil {
			fail(err)
		}
	case "killpid", "kp":
		if len(args) < 2 {
			fail(fmt.Errorf("缺少 PID，例如: kill-port-cli killpid 1234"))
		}
		pid, err := strconv.Atoi(args[1])
		if err != nil || pid <= 0 {
			fail(fmt.Errorf("无效的 PID: %s", args[1]))
		}
		if !hasFlag(args[2:], "-y", "--yes") && !confirm(fmt.Sprintf("确定要强制结束进程 %s (PID=%d) 吗?", processName(pid), pid)) {
			fmt.Println("已取消。")
			return
		}
		if err := killPID(pid); err != nil {
			fail(err)
		}
		fmt.Printf("已结束进程 %s (PID=%d)。\n", processName(pid), pid)
	case "help", "-h", "--help":
		fmt.Println(helpText)
	default:
		fmt.Printf("未知命令: %s\n\n", args[0])
		fmt.Println(helpText)
		os.Exit(2)
	}
}

// ---------------------------------------------------------------- 交互菜单

func interactiveMenu() {
	reader := bufio.NewReader(os.Stdin)
	banner()
	for {
		fmt.Println()
		fmt.Println("──────────────────────────────────────────────")
		fmt.Println("  1. 列出所有端口及占用程序")
		fmt.Println("  2. 查询指定端口被哪个程序占用")
		fmt.Println("  3. 关闭占用指定端口的程序")
		fmt.Println("  4. 按 PID 强制结束进程")
		fmt.Println("  0. 退出")
		fmt.Print("请选择> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return // EOF，直接退出
		}
		switch strings.TrimSpace(line) {
		case "1":
			if err := doList(); err != nil {
				fail(err)
			}
		case "2":
			port, err := readPort(reader)
			if err != nil {
				fmt.Println(" *", err)
				continue
			}
			conns, err := findPort(port)
			if err != nil {
				fail(err)
				continue
			}
			if len(conns) == 0 {
				fmt.Printf("端口 %d 当前没有被任何程序占用。\n", port)
			} else {
				printTable(conns)
			}
		case "3":
			port, err := readPort(reader)
			if err != nil {
				fmt.Println(" *", err)
				continue
			}
			if err := killPort(port, false); err != nil {
				fail(err)
			}
		case "4":
			fmt.Print("请输入 PID> ")
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			pid, err := strconv.Atoi(strings.TrimSpace(line))
			if err != nil || pid <= 0 {
				fmt.Println(" * 无效的 PID")
				continue
			}
			if !confirm(fmt.Sprintf("确定要强制结束进程 %s (PID=%d) 吗?", processName(pid), pid)) {
				fmt.Println("已取消。")
				continue
			}
			if err := killPID(pid); err != nil {
				fail(err)
				continue
			}
			fmt.Printf("已结束进程 %s (PID=%d)。\n", processName(pid), pid)
		case "0", "q", "quit", "exit":
			fmt.Println("再见！")
			return
		default:
			fmt.Println(" * 无效选择，请输入 0-4")
		}
	}
}

func banner() {
	fmt.Println("==============================================")
	fmt.Println("        kill-port  端口查询 / 进程终止")
	fmt.Println("==============================================")
}

func readPort(reader *bufio.Reader) (int, error) {
	fmt.Print("请输入端口号 (1-65535)> ")
	line, err := reader.ReadString('\n')
	if err != nil {
		os.Exit(0)
	}
	return parsePort(line)
}

func confirm(prompt string) bool {
	fmt.Print(prompt + " [y/N]> ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	ans := strings.ToLower(strings.TrimSpace(line))
	return ans == "y" || ans == "yes"
}

func doList() error {
	conns, err := listenConns()
	if err != nil {
		return err
	}
	if len(conns) == 0 {
		fmt.Println("未发现任何监听端口。")
		return nil
	}
	printTable(conns)
	fmt.Printf("共 %d 条监听记录。\n", len(conns))
	return nil
}

// killPort 结束占用指定端口的全部进程
func killPort(port int, autoYes bool) error {
	conns, err := findPort(port)
	if err != nil {
		return err
	}
	if len(conns) == 0 {
		fmt.Printf("端口 %d 当前没有被任何程序占用，无需关闭。\n", port)
		return nil
	}
	printTable(conns)

	seen := map[int]bool{}
	var pids []int
	for _, c := range conns {
		if c.PID > 0 && !seen[c.PID] {
			seen[c.PID] = true
			pids = append(pids, c.PID)
		}
	}
	if len(pids) == 0 {
		return fmt.Errorf("未能定位 PID（Linux/macOS 下需 root 权限才能看到其它用户的进程）")
	}
	if !autoYes && !confirm(fmt.Sprintf("确定要结束占用端口 %d 的 %d 个进程吗?", port, len(pids))) {
		fmt.Println("已取消。")
		return nil
	}
	var errs []string
	for _, pid := range pids {
		name := processName(pid)
		if err := killPID(pid); err != nil {
			errs = append(errs, fmt.Sprintf("PID=%d: %v", pid, err))
		} else {
			fmt.Printf("✔ 已结束 %s (PID=%d)\n", name, pid)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("部分进程结束失败:\n  %s\n(Windows 请尝试以管理员身份重新运行)", strings.Join(errs, "\n  "))
	}
	fmt.Printf("端口 %d 已释放。\n", port)
	return nil
}

var _ = runtime.GOOS // 保持与旧版本一致的导入占位（banner 等使用）
