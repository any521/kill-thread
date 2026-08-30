// kill-port —— 端口查询 / 端口进程终止工具
//
// 用法：
//
//	直接运行（无参数）：进入中文交互菜单
//	kill-port list                    列出所有监听端口及占用进程
//	kill-port query <端口>            查询某个端口被哪个进程占用
//	kill-port kill <端口> [-y]        关闭占用某端口的进程（-y 跳过确认）
//	kill-port killpid <PID> [-y]      按 PID 强制结束进程
//	kill-port help                    显示帮助
//
// 支持 Windows（netstat/tasklist/taskkill）与 Linux/macOS（netstat/ss + kill）。
package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Conn 表示一条本地监听/占用中的端口
type Conn struct {
	Proto string // TCP / UDP
	Addr  string // 本地地址，如 0.0.0.0:8080
	Port  int    // 端口号
	PID   int    // 进程 PID
	Name  string // 进程名
}

const helpText = `kill-port —— 端口查询 / 端口进程终止工具

用法:
  kill-port                 进入交互菜单
  kill-port list            列出所有监听端口及占用进程
  kill-port query <端口>    查询某端口被哪个进程占用
  kill-port kill <端口>     结束占用某端口的进程   (加 -y 跳过确认)
  kill-port killpid <PID>   按 PID 强制结束进程     (加 -y 跳过确认)
  kill-port help            显示本帮助

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
			fail(fmt.Errorf("缺少端口号，例如: kill-port query 8080"))
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
			fail(fmt.Errorf("缺少端口号，例如: kill-port kill 8080"))
		}
		port, err := parsePort(args[1])
		if err != nil {
			fail(err)
		}
		autoYes := hasFlag(args[2:], "-y", "--yes")
		if err := killPort(port, autoYes); err != nil {
			fail(err)
		}
	case "killpid", "kp":
		if len(args) < 2 {
			fail(fmt.Errorf("缺少 PID，例如: kill-port killpid 1234"))
		}
		pid, err := strconv.Atoi(args[1])
		if err != nil || pid <= 0 {
			fail(fmt.Errorf("无效的 PID: %s", args[1]))
		}
		autoYes := hasFlag(args[2:], "-y", "--yes")
		name := processName(pid)
		if !autoYes && !confirm(fmt.Sprintf("确定要强制结束进程 %s (PID=%d) 吗?", name, pid)) {
			fmt.Println("已取消。")
			return
		}
		if err := killPID(pid); err != nil {
			fail(err)
		}
		fmt.Printf("已结束进程 %s (PID=%d)。\n", name, pid)
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
			name := processName(pid)
			if !confirm(fmt.Sprintf("确定要强制结束进程 %s (PID=%d) 吗?", name, pid)) {
				fmt.Println("已取消。")
				continue
			}
			if err := killPID(pid); err != nil {
				fail(err)
				continue
			}
			fmt.Printf("已结束进程 %s (PID=%d)。\n", name, pid)
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

func parsePort(s string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 1 || n > 65535 {
		return 0, fmt.Errorf("无效端口号: %s（应为 1-65535）", strings.TrimSpace(s))
	}
	return n, nil
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

// ---------------------------------------------------------------- 核心操作

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

func findPort(port int) ([]Conn, error) {
	conns, err := listenConns()
	if err != nil {
		return nil, err
	}
	var out []Conn
	for _, c := range conns {
		if c.Port == port {
			out = append(out, c)
		}
	}
	return out, nil
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

func printTable(conns []Conn) {
	fmt.Println()
	fmt.Printf("%-6s %-26s %-8s %-8s %s\n", "协议", "本地地址", "端口", "PID", "进程名")
	fmt.Println(strings.Repeat("─", 76))
	for _, c := range conns {
		addr := c.Addr
		if len(addr) > 26 {
			addr = addr[:26]
		}
		fmt.Printf("%-6s %-26s %-8d %-8d %s\n", c.Proto, addr, c.Port, c.PID, c.Name)
	}
}

// ---------------------------------------------------------------- 平台实现

func listenConns() ([]Conn, error) {
	if runtime.GOOS == "windows" {
		return windowsListenConns()
	}
	return unixListenConns()
}

func killPID(pid int) error {
	if runtime.GOOS == "windows" {
		out, err := exec.Command("taskkill", "/F", "/PID", strconv.Itoa(pid)).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%v (%s)", err, strings.TrimSpace(string(out)))
		}
		time.Sleep(500 * time.Millisecond) // 等待端口释放
		return nil
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := p.Signal(syscall.SIGKILL); err != nil {
		return err
	}
	time.Sleep(200 * time.Millisecond)
	return nil
}

func processName(pid int) string {
	if pid <= 0 {
		return "?"
	}
	if runtime.GOOS == "windows" {
		names, err := windowsProcNames()
		if err == nil {
			if n, ok := names[pid]; ok {
				return n
			}
		}
		return "?"
	}
	if b, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid)); err == nil {
		return strings.TrimSpace(string(b))
	}
	return "?"
}

// ---------------- Windows ----------------

// windowsListenConns 解析 netstat -ano 输出:
//
//	TCP  0.0.0.0:135  0.0.0.0:0  LISTENING  1024
//	UDP  0.0.0.0:5353 *:*                   4
func windowsListenConns() ([]Conn, error) {
	out, err := exec.Command("netstat", "-ano").Output()
	if err != nil {
		return nil, fmt.Errorf("执行 netstat -ano 失败: %w", err)
	}
	names, _ := windowsProcNames()
	var conns []Conn
	re := regexp.MustCompile(`^(TCP|UDP)\s+(\S+)\s+\S+\s+(LISTENING\s+)?(\d+)\s*$`)
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimRight(line, "\r")
		m := re.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		proto, addr, pidStr := m[1], m[2], m[4]
		if proto == "TCP" && m[3] == "" { // 只保留 LISTENING
			continue
		}
		port, err := portOf(addr)
		if err != nil {
			continue
		}
		pid, _ := strconv.Atoi(pidStr)
		name := names[pid]
		if name == "" {
			name = "?"
		}
		conns = append(conns, Conn{Proto: proto, Addr: addr, Port: port, PID: pid, Name: name})
	}
	sortConns(conns)
	return conns, nil
}

// windowsProcNames 通过 tasklist /FO CSV /NH 得到 PID -> 映像名
func windowsProcNames() (map[int]string, error) {
	out, err := exec.Command("tasklist", "/FO", "CSV", "/NH").Output()
	if err != nil {
		return nil, fmt.Errorf("执行 tasklist 失败: %w", err)
	}
	records, err := csv.NewReader(strings.NewReader(string(out))).ReadAll()
	if err != nil {
		return nil, err
	}
	names := map[int]string{}
	for _, rec := range records {
		if len(rec) < 2 {
			continue
		}
		if pid, err := strconv.Atoi(strings.TrimSpace(rec[1])); err == nil {
			names[pid] = rec[0]
		}
	}
	return names, nil
}

// ---------------- Linux / macOS ----------------

func unixListenConns() ([]Conn, error) {
	conns, err := unixNetstat()
	if err == nil && len(conns) > 0 {
		return conns, nil
	}
	return unixSS()
}

// unixNetstat 解析 netstat -tulnp:
//
//	tcp  0  0 0.0.0.0:22  0.0.0.0:*  LISTEN  1000/sshd
func unixNetstat() ([]Conn, error) {
	out, err := exec.Command("netstat", "-tulnp").Output()
	if err != nil && len(out) == 0 {
		return nil, err
	}
	var conns []Conn
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) < 6 || (!strings.HasPrefix(f[0], "tcp") && !strings.HasPrefix(f[0], "udp")) {
			continue
		}
		proto := strings.ToUpper(f[0][:3])
		if proto == "TCP" && f[5] != "LISTEN" {
			continue
		}
		port, err := portOf(f[3])
		if err != nil {
			continue
		}
		pid, name := 0, "?"
		if ln := f[len(f)-1]; ln != "-" && ln != "*" {
			if i := strings.Index(ln, "/"); i > 0 {
				pid, _ = strconv.Atoi(ln[:i])
				name = ln[i+1:]
			}
		}
		conns = append(conns, Conn{Proto: proto, Addr: f[3], Port: port, PID: pid, Name: name})
	}
	sortConns(conns)
	return conns, nil
}

// unixSS 解析 ss -H -ltnp / ss -H -lunp:
//
//	LISTEN 0 128 0.0.0.0:22 0.0.0.0:* users:(("sshd",pid=1000,fd=3))
func unixSS() ([]Conn, error) {
	var conns []Conn
	rePID := regexp.MustCompile(`pid=(\d+)`)
	reName := regexp.MustCompile(`\(\("([^"]+)"`)
	for _, flags := range [][]string{{"-H", "-ltnp"}, {"-H", "-lunp"}} {
		out, err := exec.Command("ss", flags...).Output()
		if err != nil && len(out) == 0 {
			continue
		}
		proto := "TCP"
		if strings.Contains(flags[1], "u") {
			proto = "UDP"
		}
		for _, line := range strings.Split(string(out), "\n") {
			f := strings.Fields(line)
			if len(f) < 4 {
				continue
			}
			port, err := portOf(f[3])
			if err != nil {
				continue
			}
			pid, name := 0, "?"
			if m := rePID.FindStringSubmatch(line); m != nil {
				pid, _ = strconv.Atoi(m[1])
			}
			if m := reName.FindStringSubmatch(line); m != nil {
				name = m[1]
			}
			conns = append(conns, Conn{Proto: proto, Addr: f[3], Port: port, PID: pid, Name: name})
		}
	}
	sortConns(conns)
	return conns, nil
}

// ---------------------------------------------------------------- 工具函数

// portOf 从 "0.0.0.0:8080" / "[::]:443" / "*:53" 提取端口号
func portOf(addr string) (int, error) {
	i := strings.LastIndex(addr, ":")
	if i < 0 || i == len(addr)-1 {
		return 0, fmt.Errorf("bad addr")
	}
	return strconv.Atoi(addr[i+1:])
}

func sortConns(conns []Conn) {
	sort.SliceStable(conns, func(i, j int) bool {
		if conns[i].Port != conns[j].Port {
			return conns[i].Port < conns[j].Port
		}
		return conns[i].Proto < conns[j].Proto
	})
}

func hasFlag(args []string, flags ...string) bool {
	for _, a := range args {
		for _, f := range flags {
			if a == f {
				return true
			}
		}
	}
	return false
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, " ✘", err)
	os.Exit(1)
}
