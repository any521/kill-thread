package main

import (
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

// listenConns 返回当前所有监听端口及占用进程
func listenConns() ([]Conn, error) {
	if runtime.GOOS == "windows" {
		return windowsListenConns()
	}
	return unixListenConns()
}

// findPort 查询占用指定端口的连接
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

func killPID(pid int) error {
	if runtime.GOOS == "windows" {
		tcmd := exec.Command("taskkill", "/F", "/PID", strconv.Itoa(pid))
		tcmd.SysProcAttr = hideWindowAttr()
		out, err := tcmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%v (%s)", err, strings.TrimSpace(string(out)))
		}
		time.Sleep(300 * time.Millisecond) // 等待端口释放
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
	cmd := exec.Command("netstat", "-ano")
	cmd.SysProcAttr = hideWindowAttr()
	out, err := cmd.Output()
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
	tcmd := exec.Command("tasklist", "/FO", "CSV", "/NH")
	tcmd.SysProcAttr = hideWindowAttr()
	out, err := tcmd.Output()
	if err != nil {
		return nil, fmt.Errorf("执行 tasklist 失败: %w", err)
	}
	records, err := csvReadAll(out)
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
		scmd := exec.Command("ss", flags...)
		scmd.SysProcAttr = hideWindowAttr()
		out, err := scmd.Output()
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

// ---------------- 工具 ----------------

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

func parsePort(s string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 1 || n > 65535 {
		return 0, fmt.Errorf("无效端口号: %s（应为 1-65535）", strings.TrimSpace(s))
	}
	return n, nil
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

func fail(err error) {
	fmt.Fprintln(os.Stderr, " ✘", err)
	os.Exit(1)
}
