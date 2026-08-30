//go:build gui

// kill-port 图形界面版（Windows）：纯 syscall Win32 API 实现，零第三方依赖。
// 编译: go build -tags gui -ldflags "-s -w -H windowsgui" -o kill-port.exe .
package main

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	comctl32 = syscall.NewLazyDLL("comctl32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")

	registerClassExW = user32.NewProc("RegisterClassExW")
	createWindowExW  = user32.NewProc("CreateWindowExW")
	showWindow       = user32.NewProc("ShowWindow")
	updateWindow     = user32.NewProc("UpdateWindow")
	getMessageW      = user32.NewProc("GetMessageW")
	translateMessage = user32.NewProc("TranslateMessage")
	dispatchMessageW = user32.NewProc("DispatchMessageW")
	defWindowProcW   = user32.NewProc("DefWindowProcW")
	postQuitMessage  = user32.NewProc("PostQuitMessage")
	destroyWindow    = user32.NewProc("DestroyWindow")
	sendMessageW     = user32.NewProc("SendMessageW")
	getClientRect    = user32.NewProc("GetClientRect")
	moveWindow       = user32.NewProc("MoveWindow")
	setTimer         = user32.NewProc("SetTimer")
	killTimer        = user32.NewProc("KillTimer")
	messageBoxW      = user32.NewProc("MessageBoxW")
	loadImageW       = user32.NewProc("LoadImageW")
	loadIconW        = user32.NewProc("LoadIconW")
	createFontW      = user32.NewProc("CreateFontW")
	getModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	initCommonCtl    = comctl32.NewProc("InitCommonControlsEx")
	isUserAdmin      = shell32.NewProc("IsUserAnAdmin")
)

const (
	WM_DESTROY                   = 0x0002
	WM_SIZE                      = 0x0005
	WM_SETTEXT                   = 0x000C
	WM_GETTEXT                   = 0x000D
	WM_COMMAND                   = 0x0111
	WM_SETFONT                   = 0x0030
	WM_TIMER                     = 0x0113
	WS_CHILD                     = 0x40000000
	WS_VISIBLE                   = 0x10000000
	WS_OVERLAPPEDWINDOW          = 0x00CF0000
	WS_BORDER                    = 0x00800000
	WS_TABSTOP                   = 0x00010000
	CW_USEDEFAULT                = 0x80000000
	SW_SHOW                      = 5
	BS_PUSHBUTTON                = 0x00000000
	BS_AUTOCHECKBOX              = 0x00000003
	ES_AUTOHSCROLL               = 0x00000080
	LVS_REPORT                   = 0x0001
	LVS_SHOWSELALWAYS            = 0x0004
	LVM_FIRST                    = 0x1000
	LVM_INSERTITEMW              = LVM_FIRST + 77
	LVM_INSERTCOLUMNW            = LVM_FIRST + 97
	LVM_DELETEALLITEMS           = LVM_FIRST + 9
	LVM_GETITEMCOUNT             = LVM_FIRST + 4
	LVM_GETNEXTITEMW             = LVM_FIRST + 12
	LVM_SETITEMTEXTW             = LVM_FIRST + 116
	LVM_SETCOLUMNWIDTH           = LVM_FIRST + 30
	LVM_SETEXTENDEDLISTVIEWSTYLE = LVM_FIRST + 54
	LVIF_TEXT                    = 0x0001
	LVCF_TEXT                    = 0x0001
	LVCF_WIDTH                   = 0x0002
	LVCF_FMT                     = 0x0004
	LVCF_SUBITEM                 = 0x0008
	LVNI_SELECTED                = 0x0002
	LVS_EX_FULLROWSELECT         = 0x00000020
	LVS_EX_GRIDLINES             = 0x00000001
	BN_CLICKED                   = 0
	EN_CHANGE                    = 0x0100
	NM_DBLCLK                    = 0xFFFD // (int)-3 的低 16 位
	BM_SETCHECK                  = 0x00F1
	BST_CHECKED                  = 1
	MB_OK                        = 0x00000000
	MB_YESNO                     = 0x00000004
	MB_ICONWARNING               = 0x00000030
	MB_ICONINFO                  = 0x00000040
	MB_DEFBUTTON2                = 0x00000100
	IDI_APPICON                  = 32512
	LR_DEFAULTCOLOR              = 0x0000
	IMAGE_ICON                   = 1
	ICC_LISTVIEW_CLASSES         = 0x00000001
	ICC_BAR_CLASSES              = 0x00000004
	ICC_STANDARD_CLASSES         = 0x00002000
)

const (
	ID_LIST         = 101
	ID_EDIT         = 102
	ID_BTN_REFRESH  = 103
	ID_BTN_KILL     = 104
	ID_BTN_KILLPORT = 105
	ID_CHK_AUTO     = 106
	ID_STATUS       = 107
)

const appTitle = "kill-port —— 端口查询 / 进程终止  v1.1.0"

type wndclassex struct {
	size         uint32
	style        uint32
	lpfnWndProc  uintptr
	cbClsExtra   int32
	cbWndExtra   int32
	hInstance    syscall.Handle
	hIcon        syscall.Handle
	hCursor      syscall.Handle
	hBackground  syscall.Handle
	lpszMenuName *uint16
	lpszClass    *uint16
	hIconSm      syscall.Handle
}

type rect struct{ l, t, r, b int32 }

type msg struct {
	hwnd     syscall.Handle
	message  uint32
	reserved uint32
	wParam   uintptr
	lParam   int32
	time     uint32
	x, y     int32
	private  uint32
}

type lvitem struct {
	mask      uint32
	iItem     int32
	iSubItem  int32
	state     uint32
	stateMask uint32
	text      *uint16
	cchMax    int32
	iImage    int32
	lParam    uintptr
}

type lvcolumn struct {
	mask     uint32
	fmt      int32
	cx       int32
	text     *uint16
	cchMax   int32
	iSubItem int32
	iImage   int32
	iOrder   int32
}

type initcommoncontrolsex struct {
	size uint32
	icc  uint32
}

var (
	hwndMain syscall.Handle
	hwndList syscall.Handle
	hwndEdit syscall.Handle
	hwndStat syscall.Handle
	hwndChk  syscall.Handle
	hwndBR1  syscall.Handle
	hwndBR2  syscall.Handle
	hwndBR3  syscall.Handle
	hFont    syscall.Handle

	allData []Conn
	shown   []Conn
	autoRef bool = true
)

func u16p(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }
func sendMsg(h syscall.Handle, m uint32, wp, lp uintptr) uintptr {
	r, _, _ := sendMessageW.Call(uintptr(h), uintptr(m), wp, lp)
	return r
}

func main() {
	runtime.LockOSThread()
	if err := runGUI(); err != nil {
		messageBoxW.Call(0, uintptr(unsafe.Pointer(u16p("启动失败: "+err.Error()))), uintptr(unsafe.Pointer(u16p("kill-port"))), MB_ICONWARNING)
	}
}

func runGUI() error {
	var hinst syscall.Handle
	h, _, _ := getModuleHandleW.Call(0)
	hinst = syscall.Handle(h)

	icc := initcommoncontrolsex{size: uint32(unsafe.Sizeof(initcommoncontrolsex{})), icc: ICC_LISTVIEW_CLASSES | ICC_BAR_CLASSES | ICC_STANDARD_CLASSES}
	initCommonCtl.Call(uintptr(unsafe.Pointer(&icc)))

	// 图标：优先用内嵌资源(group 名 "APP"，由 go-winres 打包)
	var hIcon syscall.Handle
	ri, _, _ := loadImageW.Call(uintptr(hinst), uintptr(unsafe.Pointer(u16p("APP"))), IMAGE_ICON, 0, 0, LR_DEFAULTCOLOR)
	if ri != 0 {
		hIcon = syscall.Handle(ri)
	} else {
		ri, _, _ := loadIconW.Call(0, uintptr(IDI_APPICON))
		hIcon = syscall.Handle(ri)
	}

	wc := wndclassex{}
	wc.size = uint32(unsafe.Sizeof(wc))
	wc.style = 0x0003 // CS_HREDRAW|CS_VREDRAW
	wc.lpfnWndProc = syscall.NewCallback(wndProc)
	wc.hInstance = hinst
	wc.hIcon = hIcon
	wc.hIconSm = hIcon
	cur, _, _ := user32.NewProc("LoadCursorW").Call(0, 32512 /*IDC_ARROW*/)
	wc.hCursor = syscall.Handle(cur)
	wc.hBackground = syscall.Handle(16) // (COLOR_WINDOW+1)
	wc.lpszClass = u16p("killPortGUI")
	if r, _, _ := registerClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return fmt.Errorf("RegisterClassEx 失败")
	}

	hwnd, _, _ := createWindowExW.Call(0, uintptr(unsafe.Pointer(u16p("killPortGUI"))),
		uintptr(unsafe.Pointer(u16p(appTitle))), WS_OVERLAPPEDWINDOW,
		CW_USEDEFAULT, CW_USEDEFAULT, 780, 540, 0, 0, uintptr(hinst), 0)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowEx 失败")
	}
	hwndMain = syscall.Handle(hwnd)

	createControls()

	showWindow.Call(uintptr(hwndMain), SW_SHOW)
	updateWindow.Call(uintptr(hwndMain))
	doRefresh()

	var m msg
	for {
		r, _, _ := getMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 { // 0=WM_QUIT，-1=错误
			break
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&m)))
		dispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	return nil
}

func mkCtl(class, text string, style uint32, id int) syscall.Handle {
	h, _, _ := createWindowExW.Call(0, uintptr(unsafe.Pointer(u16p(class))), uintptr(unsafe.Pointer(u16p(text))),
		uintptr(WS_CHILD|WS_VISIBLE|style), 0, 0, 10, 10, uintptr(hwndMain), uintptr(id), 0, 0)
	return syscall.Handle(h)
}

func createControls() {
	// 微软雅黑 10 号
	fontHeight := -14
	f, _, _ := createFontW.Call(uintptr(fontHeight), 0, 0, 0, 400, /*FW_NORMAL*/
		0, 0, 0, 134 /*GB2312_CHARSET*/, 0, 0, 1 /*DEFAULT_QUALITY*/, 0x20, /*FF_SWISS*/
		uintptr(unsafe.Pointer(u16p("微软雅黑"))))
	hFont = syscall.Handle(f)

	hwndBR1 = mkCtl("BUTTON", "刷新 (F5)", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_REFRESH)
	hwndBR2 = mkCtl("BUTTON", "结束选中进程", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_KILL)
	hwndBR3 = mkCtl("BUTTON", "结束该端口全部进程", BS_PUSHBUTTON|WS_TABSTOP, ID_BTN_KILLPORT)
	hwndChk = mkCtl("BUTTON", "自动刷新", BS_AUTOCHECKBOX, ID_CHK_AUTO)
	sendMsg(hwndChk, BM_SETCHECK, BST_CHECKED, 0)
	hwndEdit = mkCtl("EDIT", "", WS_BORDER|ES_AUTOHSCROLL|WS_TABSTOP, ID_EDIT)
	hwndList = mkCtl("SysListView32", "", WS_BORDER|LVS_REPORT|LVS_SHOWSELALWAYS|WS_TABSTOP, ID_LIST)
	sendMsg(hwndList, LVM_SETEXTENDEDLISTVIEWSTYLE, LVS_EX_FULLROWSELECT|LVS_EX_GRIDLINES, LVS_EX_FULLROWSELECT|LVS_EX_GRIDLINES)

	cols := []struct {
		name string
		w    int32
	}{{"协议", 60}, {"本地地址", 230}, {"端口", 64}, {"PID", 74}, {"进程名", 190}}
	var c lvcolumn
	c.mask = LVCF_TEXT | LVCF_WIDTH | LVCF_FMT | LVCF_SUBITEM
	for i, col := range cols {
		c.cx = col.w
		c.text = u16p(col.name)
		c.iSubItem = int32(i)
		c.fmt = 0 // LEFT
		sendMsg(hwndList, LVM_INSERTCOLUMNW, uintptr(i), uintptr(unsafe.Pointer(&c)))
	}

	hwndStat = mkCtl("STATIC", "就绪", 0, ID_STATUS)

	for _, h := range []syscall.Handle{hwndBR1, hwndBR2, hwndBR3, hwndChk, hwndEdit, hwndList, hwndStat} {
		sendMsg(h, WM_SETFONT, uintptr(hFont), 1)
	}
	sendMsg(hwndMain, WM_SETTEXT, 0, 0) // touch

	setTimer.Call(uintptr(hwndMain), 1, 2000, 0)
}

func layout() {
	var r rect
	getClientRect.Call(uintptr(hwndMain), uintptr(unsafe.Pointer(&r)))
	W := int(r.r)
	y := 8
	x := 8
	for _, b := range []syscall.Handle{hwndBR1, hwndBR2, hwndBR3} {
		t, _, _ := textWidthOf(b)
		moveWindow.Call(uintptr(b), uintptr(x), uintptr(y), uintptr(t), 28, 1)
		x += t + 8
	}
	tchk, _, _ := textWidthOf(hwndChk)
	moveWindow.Call(uintptr(hwndChk), uintptr(x), uintptr(y), uintptr(tchk), 28, 1)
	editW := 190
	moveWindow.Call(uintptr(hwndEdit), uintptr(W-editW-8), uintptr(y), uintptr(editW), 28, 1)
	lh := int(r.b) - 28 - 24 - 16
	moveWindow.Call(uintptr(hwndList), 8, uintptr(28+16), uintptr(W-16), uintptr(lh), 1)
	moveWindow.Call(uintptr(hwndStat), 8, uintptr(int(r.b)-22), uintptr(W-16), 20, 1)
}

// 估算控件需要的宽度（中文按 2 字节近似）
func textWidthOf(h syscall.Handle) (int, int, int) {
	buf := make([]uint16, 64)
	n, _, _ := sendMessageW.Call(uintptr(h), WM_GETTEXT, 64, uintptr(unsafe.Pointer(&buf[0])))
	ln := int(n)
	if ln > 63 {
		ln = 63
	}
	s := syscall.UTF16ToString(buf[:ln])
	w := 0
	for _, r := range s {
		if r > 0x2E80 {
			w += 16
		} else {
			w += 9
		}
	}
	return w + 24, 0, 0
}

func wndProc(h syscall.Handle, msg uint32, wp, lp uintptr) uintptr {
	switch msg {
	case WM_SIZE:
		layout()
		return 0
	case WM_TIMER:
		if wp == 1 && autoRef {
			doRefresh()
		}
		return 0
	case WM_COMMAND:
		id := wp & 0xFFFF
		code := (wp >> 16) & 0xFFFF
		switch {
		case code == BN_CLICKED && id == ID_BTN_REFRESH:
			doRefresh()
		case code == BN_CLICKED && id == ID_BTN_KILL:
			killSelected(false)
		case code == BN_CLICKED && id == ID_BTN_KILLPORT:
			killSelected(true)
		case code == BN_CLICKED && id == ID_CHK_AUTO:
			ck := sendMsg(hwndChk, 0x00F0 /*BM_GETCHECK*/, 0, 0)
			autoRef = ck == BST_CHECKED
		case code == EN_CHANGE && id == ID_EDIT:
			applyFilter()
		case code == NM_DBLCLK && id == ID_LIST:
			killSelected(false)
		}
		return 0
	case WM_DESTROY:
		killTimer.Call(uintptr(h), 1)
		postQuitMessage.Call(0)
		return 0
	}
	r, _, _ := defWindowProcW.Call(uintptr(h), uintptr(msg), wp, lp)
	return r
}

// ---------------------------------------------------------------- 数据/刷新

func doRefresh() {
	conns, err := listenConns()
	if err != nil {
		setStatus("查询失败: " + err.Error())
		return
	}
	allData = conns
	applyFilter()
}

func editString() string {
	buf := make([]uint16, 128)
	n, _, _ := sendMessageW.Call(uintptr(hwndEdit), WM_GETTEXT, 128, uintptr(unsafe.Pointer(&buf[0])))
	return strings.TrimSpace(syscall.UTF16ToString(buf[:n]))
}

func applyFilter() {
	f := strings.ToLower(editString())
	shown = shown[:0]
	for _, c := range allData {
		if f == "" ||
			strings.Contains(strings.ToLower(c.Name), f) ||
			strings.Contains(fmt.Sprintf("%d", c.Port), f) ||
			strings.Contains(strings.ToLower(c.Addr), f) ||
			strings.Contains(strings.ToLower(c.Proto), f) {
			shown = append(shown, c)
		}
	}
	rebuildList()
}

func rebuildList() {
	sendMsg(hwndList, LVM_DELETEALLITEMS, 0, 0)
	for i, c := range shown {
		cells := []string{c.Proto, c.Addr, fmt.Sprintf("%d", c.Port), fmt.Sprintf("%d", c.PID), c.Name}
		var it lvitem
		it.mask = LVIF_TEXT
		it.iItem = int32(i)
		it.iImage = -1
		it.text = u16p(cells[0])
		sendMsg(hwndList, LVM_INSERTITEMW, 0, uintptr(unsafe.Pointer(&it)))
		for col := 1; col < 5; col++ {
			it.iSubItem = int32(col)
			it.text = u16p(cells[col])
			sendMsg(hwndList, LVM_SETITEMTEXTW, uintptr(i), uintptr(unsafe.Pointer(&it)))
		}
	}
	setStatus(fmt.Sprintf("显示 %d / 共 %d 条   刷新于 %s   %s",
		len(shown), len(allData), time.Now().Format("15:04:05"), adminHint()))
}

func adminHint() string {
	a, _, _ := isUserAdmin.Call()
	if a == 0 {
		return "（当前非管理员：系统/他人进程可能无法结束）"
	}
	return "（管理员权限）"
}

func setStatus(s string) {
	sendMsg(hwndStat, WM_SETTEXT, 0, uintptr(unsafe.Pointer(u16p(s))))
}

// selectedRows 返回选中行索引
// selectedRows 返回选中行索引（LVM_GETNEXTITEM 游标式遍历）
func selectedRows() []int {
	var out []int
	cur := ^uintptr(0) // -1 起点
	for {
		r, _, _ := sendMessageW.Call(uintptr(hwndList), LVM_GETNEXTITEMW, cur, LVNI_SELECTED)
		ni := int32(r)
		if ni < 0 {
			break
		}
		out = append(out, int(ni))
		cur = uintptr(ni)
	}
	return out
}

func messageBox(text, title string, flags uintptr) uintptr {
	r, _, _ := messageBoxW.Call(uintptr(hwndMain), uintptr(unsafe.Pointer(u16p(text))), uintptr(unsafe.Pointer(u16p(title))), flags)
	return r
}

// ---------------------------------------------------------------- 终止操作

// killSelected 结束选中进程；byPort=true 时结束选中行对应端口的全部进程
func killSelected(byPort bool) {
	rows := selectedRows()
	if len(rows) == 0 {
		messageBox("请先在列表中选择至少一行。", "kill-port", MB_ICONINFO)
		return
	}
	type target struct {
		pid  int
		name string
		port int
	}
	var targets []target
	seen := map[int]bool{}
	if byPort {
		ports := map[int]bool{}
		for _, r := range rows {
			ports[shown[r].Port] = true
		}
		for _, c := range allData {
			if ports[c.Port] && c.PID > 0 && !seen[c.PID] {
				seen[c.PID] = true
				targets = append(targets, target{c.PID, c.Name, c.Port})
			}
		}
		// 按 PID 排序，保证展示顺序稳定
		sort.Slice(targets, func(i, j int) bool { return targets[i].pid < targets[j].pid })
	} else {
		for _, r := range rows {
			c := shown[r]
			if c.PID > 0 && !seen[c.PID] {
				seen[c.PID] = true
				targets = append(targets, target{c.PID, c.Name, c.Port})
			}
		}
	}
	if len(targets) == 0 {
		messageBox("选中行没有可用的 PID。", "kill-port", MB_ICONINFO)
		return
	}
	lines := make([]string, 0, len(targets)+1)
	if byPort {
		lines = append(lines, "将结束所选端口占用的全部进程：")
	} else {
		lines = append(lines, "确定要结束以下进程吗？")
	}
	for i, t := range targets {
		if i >= 6 {
			lines = append(lines, fmt.Sprintf("… 等共 %d 个", len(targets)))
			break
		}
		lines = append(lines, fmt.Sprintf("  %s  (PID %d, 端口 %d)", t.name, t.pid, t.port))
	}
	if messageBox(strings.Join(lines, "\n"), "确认结束进程", MB_YESNO|MB_ICONWARNING|MB_DEFBUTTON2) != 6 /*IDYES*/ {
		return
	}
	var ok, bad []string
	for _, t := range targets {
		if err := killPID(t.pid); err != nil {
			bad = append(bad, fmt.Sprintf("%s (PID %d): %v", t.name, t.pid, err))
		} else {
			ok = append(ok, fmt.Sprintf("%s (PID %d)", t.name, t.pid))
		}
	}
	doRefresh()
	var sb strings.Builder
	if len(ok) > 0 {
		sb.WriteString(fmt.Sprintf("已结束 %d 个进程:\n  ", len(ok)) + strings.Join(ok, "\n  "))
	}
	if len(bad) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(fmt.Sprintf("失败 %d 个:\n  ", len(bad)) + strings.Join(bad, "\n  ") + "\n\n提示: 以管理员身份运行后重试")
	}
	messageBox(sb.String(), "kill-port 结果", MB_ICONINFO)
}
