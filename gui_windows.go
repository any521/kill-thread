//go:build gui

// kill-port 图形界面版（Windows）—— 微信风格「端口管理」页面，纯 syscall + GDI 自绘，零依赖。
// 编译: go build -tags gui -ldflags "-s -w -H windowsgui" -o kill-port.exe .
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
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
	getClientRect    = user32.NewProc("GetClientRect")
	setTimer         = user32.NewProc("SetTimer")
	killTimer        = user32.NewProc("KillTimer")
	messageBoxW      = user32.NewProc("MessageBoxW")
	loadImageW       = user32.NewProc("LoadImageW")
	loadIconW        = user32.NewProc("LoadIconW")
	loadCursorW      = user32.NewProc("LoadCursorW")
	setCursor        = user32.NewProc("SetCursor")
	beginPaint       = user32.NewProc("BeginPaint")
	endPaint         = user32.NewProc("EndPaint")
	fillRect         = user32.NewProc("FillRect")
	drawTextW        = user32.NewProc("DrawTextW")
	invalidateRect   = user32.NewProc("InvalidateRect")
	postMessageW     = user32.NewProc("PostMessageW")

	createFontW            = gdi32.NewProc("CreateFontW") // GDI 函数在 gdi32.dll
	createSolidBrush       = gdi32.NewProc("CreateSolidBrush")
	createPen              = gdi32.NewProc("CreatePen")
	selectObject           = gdi32.NewProc("SelectObject")
	deleteObject           = gdi32.NewProc("DeleteObject")
	deleteDC               = gdi32.NewProc("DeleteDC")
	createCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	createCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	bitBlt                 = gdi32.NewProc("BitBlt")
	setBkMode              = gdi32.NewProc("SetBkMode")
	setTextColor           = gdi32.NewProc("SetTextColor")
	roundRectProc          = gdi32.NewProc("RoundRect")
	ellipseProc            = gdi32.NewProc("Ellipse")
	getStockObject         = gdi32.NewProc("GetStockObject")
	getTextExtentPoint32W  = gdi32.NewProc("GetTextExtentPoint32W")
	moveToEx               = gdi32.NewProc("MoveToEx")
	lineTo                 = gdi32.NewProc("LineTo")

	getModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	initCommonCtl    = comctl32.NewProc("InitCommonControlsEx")
	isUserAdmin      = shell32.NewProc("IsUserAnAdmin")
)

const (
	WM_KEYDOWN       = 0x0100
	WM_TIMER         = 0x0113
	WM_GETMINMAXINFO = 0x0024
	WM_PAINT         = 0x000F
	WM_ERASEBKGND    = 0x0014
	WM_SIZE          = 0x0005
	WM_SETCURSOR     = 0x0020
	WM_DESTROY       = 0x0002
	WM_MOUSEMOVE     = 0x0200
	WM_LBUTTONDOWN   = 0x0201
	WM_LBUTTONDBLCLK = 0x0203
	WM_MOUSEWHEEL    = 0x020A
	WM_CHAR          = 0x0102
	WM_APP           = 0x8000

	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	CW_USEDEFAULT       = 0x80000000
	SW_SHOW             = 5

	DT_LEFT         = 0x0000
	DT_CENTER       = 0x0001
	DT_RIGHT        = 0x0002
	DT_VCENTER      = 0x0004
	DT_SINGLELINE   = 0x0020
	DT_END_ELLIPSIS = 0x8000
	DT_NOPREFIX     = 0x0800

	TRANSPARENT = 1
	PS_SOLID    = 0
	SRCCOPY     = 0x00CC0020
	NULL_PEN    = 8
	IDI_APPICON = 32512
	IDC_ARROW   = 32512
	IDC_HAND    = 32649
	IMAGE_ICON  = 1
	VK_BACK     = 8
	VK_ESCAPE   = 27

	MB_YESNO       = 4
	MB_ICONWARNING = 0x30
	MB_ICONINFO    = 0x40
	MB_DEFBUTTON2  = 0x100

	ICC_STANDARD_CLASSES = 0x00002000
)

const appTitle = "端口管理 —— kill-port v1.2.0"

// 微信配色 (COLORREF)
func cref(r, g, b int) uintptr { return uintptr(r | g<<8 | b<<16) }

var (
	cPage        = cref(0xEF, 0xF1, 0xF2)
	cSide        = cref(0x2E, 0x2E, 0x2E)
	cCard        = cref(255, 255, 255)
	cText        = cref(0x19, 0x19, 0x19)
	cSub         = cref(0xB2, 0xB2, 0xB2)
	cGreen       = cref(0x07, 0xC1, 0x60)
	cGreenBg     = cref(0xE9, 0xF7, 0xEF)
	cGreenDis    = cref(0xAA, 0xE2, 0xC5)
	cGreenHov    = cref(0x06, 0xAD, 0x56)
	cRed         = cref(0xFA, 0x51, 0x51)
	cRedBg       = cref(0xFE, 0xF0, 0xF0)
	cRedBorder   = cref(0xFB, 0xDE, 0xDE)
	cHover       = cref(0xF7, 0xF7, 0xF7)
	cLine        = cref(0xF0, 0xF0, 0xF0)
	cBorder      = cref(0xE3, 0xE5, 0xE6)
	cOrange      = cref(0xFA, 0x8C, 0x16)
	cOrangeBg    = cref(0xFF, 0xF3, 0xE0)
	cSwitchOff   = cref(0xD0, 0xD0, 0xD0)
	cScrollBar   = cref(0xD9, 0xD9, 0xD9)
	cGrayHov     = cref(0xF2, 0xF2, 0xF2)
	cPlaceholder = cref(0xC8, 0xC8, 0xC8)
)

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

type rectT struct{ l, t, r, b int32 }
type pointT struct{ x, y int32 }
type sizeT struct{ cx, cy int32 }

type paintstruct struct {
	hdc      syscall.Handle
	fErase   uint32
	rc       rectT
	fRestore uint32
	fIgnore  uint32
}

type minmaxinfo struct {
	preserved pointT
	maxSize   pointT
	maxPos    pointT
	minTrack  pointT
	maxTrack  pointT
}

type msg struct {
	hwnd     syscall.Handle
	message  uint32
	reserved uint32
	wParam   uintptr
	lParam   uintptr
	time     uint32
	x, y     int32
	private  uint32
}

type initcommoncontrolsex struct {
	size uint32
	icc  uint32
}

const (
	hitRow = 1 + iota
	hitRowKill
	hitBtnRefresh
	hitBtnKillPort
	hitToggle
	hitSearch
	hitClear
)

type hitZone struct {
	kind int
	idx  int
	r    rectT
}

type kTarget struct {
	pid  int
	name string
	port int
}

var (
	hwndMain                        syscall.Handle
	fH1, fBig, fTx, fBd, fSm, fLogo syscall.Handle

	mu         sync.Mutex
	allData    []Conn
	pendAll    []Conn
	pendErr    error
	shown      []Conn
	refreshing bool
	killBusy   bool
	pendOk     []string
	pendBad    []string

	q         []rune
	focused   bool
	caretOn   bool
	selected  = -1
	scrollY   int
	autoOn    = true
	hits      []hitZone
	hover     hitZone
	lastTime  string
	statusMsg string
	gcl       rectT
)

func u16p(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }
func u16len(s string) int   { return len(syscall.StringToUTF16(s)) - 1 }
func gpx(lp uintptr) int    { return int(int16(uint16(lp & 0xFFFF))) }
func gpy(lp uintptr) int    { return int(int16(uint16((lp >> 16) & 0xFFFF))) }

func logf(format string, a ...any) {
	s := fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, a...))
	if f, err := os.OpenFile(filepath.Join(os.TempDir(), "kill-port.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		f.WriteString(s)
		f.Close()
	}
}

func fatal(m string) {
	logf("FATAL: %s", m)
	messageBoxW.Call(0, uintptr(unsafe.Pointer(u16p("启动失败:\n"+m))), uintptr(unsafe.Pointer(u16p("kill-port"))), MB_ICONWARNING)
	os.Exit(1)
}

func main() {
	runtime.LockOSThread()
	defer func() {
		if r := recover(); r != nil {
			fatal(fmt.Sprintf("程序异常: %v", r))
		}
	}()
	exe, _ := os.Executable()
	logf("=== 启动 v1.2.0 gui(微信风格), exe=%s ===", exe)
	if err := runGUI(); err != nil {
		fatal(err.Error())
	}
	logf("=== 正常退出 ===")
}

func runGUI() error {
	h, _, _ := getModuleHandleW.Call(0)
	hinst := syscall.Handle(h)

	icc := initcommoncontrolsex{size: uint32(unsafe.Sizeof(initcommoncontrolsex{})), icc: ICC_STANDARD_CLASSES}
	initCommonCtl.Call(uintptr(unsafe.Pointer(&icc)))

	var hIcon syscall.Handle
	if ri, _, _ := loadImageW.Call(uintptr(hinst), uintptr(unsafe.Pointer(u16p("APP"))), IMAGE_ICON, 0, 0, 0); ri != 0 {
		hIcon = syscall.Handle(ri)
	} else {
		ri, _, _ := loadIconW.Call(0, uintptr(IDI_APPICON))
		hIcon = syscall.Handle(ri)
	}

	wc := wndclassex{}
	wc.size = uint32(unsafe.Sizeof(wc))
	wc.style = 0x0003
	wc.lpfnWndProc = syscall.NewCallback(wndProc)
	wc.hInstance = hinst
	wc.hIcon = hIcon
	wc.hIconSm = hIcon
	cur, _, _ := loadCursorW.Call(0, uintptr(IDC_ARROW))
	wc.hCursor = syscall.Handle(cur)
	wc.lpszClass = u16p("killPortWX")
	if r, _, _ := registerClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return fmt.Errorf("RegisterClassEx 失败(错误码 %d)", syscall.GetLastError())
	}

	hwnd, _, _ := createWindowExW.Call(0, uintptr(unsafe.Pointer(u16p("killPortWX"))),
		uintptr(unsafe.Pointer(u16p(appTitle))), WS_OVERLAPPEDWINDOW|WS_VISIBLE,
		CW_USEDEFAULT, CW_USEDEFAULT, 920, 640, 0, 0, uintptr(hinst), 0)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowEx 失败(错误码 %d)", syscall.GetLastError())
	}
	hwndMain = syscall.Handle(hwnd)

	fH1 = mkFont(22, 700)
	fBig = mkFont(17, 700)
	fTx = mkFont(14, 400)
	fBd = mkFont(14, 700)
	fSm = mkFont(12, 400)
	fLogo = mkFont(20, 700)
	logf("窗口与字体创建 OK")

	showWindow.Call(hwnd, SW_SHOW)
	updateWindow.Call(hwnd)

	postMessageW.Call(uintptr(hwndMain), WM_APP+1, 0, 0)
	setTimer.Call(uintptr(hwndMain), 1, 2000, 0)
	setTimer.Call(uintptr(hwndMain), 2, 530, 0)

	var m msg
	for {
		r, _, _ := getMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&m)))
		dispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	return nil
}

func mkFont(px, weight int) syscall.Handle {
	h := -px
	f, _, _ := createFontW.Call(uintptr(h), 0, 0, 0, uintptr(weight),
		0, 0, 0, 134, 0, 0, 1, 0x20, uintptr(unsafe.Pointer(u16p("微软雅黑"))))
	return syscall.Handle(f)
}

// ---------------- 数据（后台线程，UI 不卡顿） ----------------

func requestRefresh() {
	mu.Lock()
	if refreshing {
		mu.Unlock()
		return
	}
	refreshing = true
	mu.Unlock()
	go func() {
		c, err := listenConns()
		mu.Lock()
		pendAll, pendErr = c, err
		refreshing = false
		mu.Unlock()
		postMessageW.Call(uintptr(hwndMain), WM_APP+2, 0, 0)
	}()
}

func startKill(targets []kTarget, desc string) {
	killBusy = true
	statusMsg = "正在结束 " + desc + " …"
	invalidate()
	go func() {
		var ok, bad []string
		for _, t := range targets {
			if err := killPID(t.pid); err != nil {
				bad = append(bad, fmt.Sprintf("%s (PID %d): %v", t.name, t.pid, err))
			} else {
				ok = append(ok, fmt.Sprintf("%s (PID %d)", t.name, t.pid))
			}
		}
		mu.Lock()
		pendOk, pendBad = ok, bad
		mu.Unlock()
		postMessageW.Call(uintptr(hwndMain), WM_APP+3, 0, 0)
	}()
}

func applyFilter() {
	key := strings.ToLower(string(q))
	shown = shown[:0]
	for _, c := range allData {
		if key == "" ||
			strings.Contains(strings.ToLower(c.Name), key) ||
			strings.Contains(fmt.Sprintf("%d", c.Port), key) ||
			strings.Contains(strings.ToLower(c.Addr), key) ||
			strings.Contains(strings.ToLower(c.Proto), key) {
			shown = append(shown, c)
		}
	}
	if selected >= len(shown) {
		selected = -1
	}
}

func invalidate() {
	invalidateRect.Call(uintptr(hwndMain), 0, 0, 1)
}

func clampScroll() {
	maxScroll := len(shown)*64 - contentH()
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scrollY > maxScroll {
		scrollY = maxScroll
	}
	if scrollY < 0 {
		scrollY = 0
	}
}

// ---------------- 窗口过程 ----------------

func wndProc(h syscall.Handle, msgid uint32, wp, lp uintptr) uintptr {
	switch msgid {
	case WM_GETMINMAXINFO:
		lpv := lp
		mm := *(**minmaxinfo)(unsafe.Pointer(&lpv))
		mm.minTrack = pointT{780, 520}
		return 0
	case WM_ERASEBKGND:
		return 1
	case WM_SIZE:
		invalidate()
		return 0
	case WM_PAINT:
		var ps paintstruct
		hdc, _, _ := beginPaint.Call(uintptr(h), uintptr(unsafe.Pointer(&ps)))
		paintAll(syscall.Handle(hdc))
		endPaint.Call(uintptr(h), uintptr(unsafe.Pointer(&ps)))
		return 0
	case WM_TIMER:
		if wp == 1 && autoOn && !killBusy {
			requestRefresh()
		}
		if wp == 2 && focused {
			caretOn = !caretOn
			invalidate()
		}
		return 0
	case WM_APP + 1:
		requestRefresh()
		return 0
	case WM_APP + 2:
		mu.Lock()
		var errText error
		allData = pendAll
		errText = pendErr
		pendAll = nil
		mu.Unlock()
		if errText != nil {
			statusMsg = "查询失败: " + errText.Error()
			logf("listenConns 失败: %v", errText)
		} else {
			statusMsg = ""
		}
		applyFilter()
		clampScroll()
		lastTime = time.Now().Format("15:04:05")
		invalidate()
		return 0
	case WM_APP + 3:
		mu.Lock()
		ok, bad := pendOk, pendBad
		pendOk, pendBad = nil, nil
		mu.Unlock()
		killBusy = false
		statusMsg = ""
		requestRefresh()
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
		if sb.Len() > 0 {
			messageBoxW.Call(uintptr(h), uintptr(unsafe.Pointer(u16p(sb.String()))), uintptr(unsafe.Pointer(u16p("kill-port 结果"))), MB_ICONINFO)
		}
		return 0
	case WM_MOUSEMOVE:
		onMove(gpx(lp), gpy(lp))
		return 0
	case WM_LBUTTONDOWN:
		onClick(gpx(lp), gpy(lp), false)
		return 0
	case WM_LBUTTONDBLCLK:
		onClick(gpx(lp), gpy(lp), true)
		return 0
	case WM_MOUSEWHEEL:
		delta := int16(uint16((wp >> 16) & 0xFFFF))
		scrollY -= int(delta) / 120 * 64
		clampScroll()
		invalidate()
		return 0
	case WM_CHAR:
		if focused {
			r := rune(wp)
			switch {
			case r == 13 || r == 27:
				focused = false
				caretOn = false
			case r == 8:
				if len(q) > 0 {
					q = q[:len(q)-1]
					applyFilter()
					clampScroll()
				}
			case r >= 32 && r < 127 && len(q) < 32:
				q = append(q, r)
				applyFilter()
				clampScroll()
			}
			caretOn = true
			invalidate()
		}
		return 0
	case WM_SETCURSOR:
		switch hover.kind {
		case hitRowKill, hitBtnRefresh, hitBtnKillPort, hitToggle, hitClear:
			hand, _, _ := loadCursorW.Call(0, uintptr(IDC_HAND))
			setCursor.Call(hand)
			return 1
		case hitRow, hitSearch:
			arrow, _, _ := loadCursorW.Call(0, uintptr(IDC_ARROW))
			setCursor.Call(arrow)
			return 1
		}
	case WM_DESTROY:
		killTimer.Call(uintptr(h), 1)
		killTimer.Call(uintptr(h), 2)
		postQuitMessage.Call(0)
		return 0
	}
	r, _, _ := defWindowProcW.Call(uintptr(h), uintptr(msgid), wp, lp)
	return r
}

// ---------------- 交互 ----------------

func hitAt(x, y int) hitZone {
	for _, pr := range []int{hitRowKill, hitClear, hitToggle, hitBtnRefresh, hitBtnKillPort, hitSearch, hitRow} {
		for i := len(hits) - 1; i >= 0; i-- {
			hz := hits[i]
			if hz.kind == pr && inRect(x, y, hz.r) {
				return hz
			}
		}
	}
	return hitZone{}
}

func inRect(x, y int, r rectT) bool {
	return int(r.l) <= x && x < int(r.r) && int(r.t) <= y && y < int(r.b)
}

func onMove(x, y int) {
	z := hitAt(x, y)
	if z.kind != hover.kind || z.idx != hover.idx {
		hover = z
		invalidate()
	}
}

func onClick(x, y int, dbl bool) {
	z := hitAt(x, y)
	focused = z.kind == hitSearch
	switch z.kind {
	case hitSearch:
		caretOn = true
	case hitClear:
		q = q[:0]
		applyFilter()
		clampScroll()
	case hitToggle:
		autoOn = !autoOn
	case hitBtnRefresh:
		requestRefresh()
	case hitBtnKillPort:
		killPortOfSelection()
	case hitRow:
		selected = z.idx
		if dbl {
			killRowAt(z.idx)
		}
	case hitRowKill:
		killRowAt(z.idx)
	default:
		if !dbl {
			selected = -1
		}
	}
	invalidate()
}

func boxWarn(s string) {
	messageBoxW.Call(uintptr(hwndMain), uintptr(unsafe.Pointer(u16p(s))), uintptr(unsafe.Pointer(u16p("kill-port"))), MB_ICONINFO)
}

func killRowAt(idx int) {
	if idx < 0 || idx >= len(shown) || killBusy {
		return
	}
	c := shown[idx]
	if c.PID <= 0 {
		boxWarn("该行没有可用的 PID。")
		return
	}
	task := fmt.Sprintf("进程 %s (PID %d)\n端口: %s %d\n\n确定结束该进程？", c.Name, c.PID, c.Proto, c.Port)
	if r, _, _ := messageBoxW.Call(uintptr(hwndMain), uintptr(unsafe.Pointer(u16p(task))), uintptr(unsafe.Pointer(u16p("确认结束进程"))), MB_YESNO|MB_ICONWARNING|MB_DEFBUTTON2); r != 6 {
		return
	}
	startKill([]kTarget{{c.PID, c.Name, c.Port}}, c.Name)
}

func killPortOfSelection() {
	if killBusy {
		return
	}
	if selected < 0 || selected >= len(shown) {
		boxWarn("请先点击列表中选择一个端口行，再结束该端口的全部进程。")
		return
	}
	port := shown[selected].Port
	seen := map[int]bool{}
	var ts []kTarget
	for _, c := range allData {
		if c.Port == port && c.PID > 0 && !seen[c.PID] {
			seen[c.PID] = true
			ts = append(ts, kTarget{c.PID, c.Name, c.Port})
		}
	}
	sort.Slice(ts, func(i, j int) bool { return ts[i].pid < ts[j].pid })
	if len(ts) == 0 {
		boxWarn("该端口没有可用的进程 PID。")
		return
	}
	lines := make([]string, 0, len(ts)+1)
	lines = append(lines, fmt.Sprintf("将结束端口 %d 的全部 %d 个占用进程:", port, len(ts)))
	for i, t := range ts {
		if i >= 6 {
			lines = append(lines, fmt.Sprintf("… 共 %d 个", len(ts)))
			break
		}
		lines = append(lines, fmt.Sprintf("  %s (PID %d)", t.name, t.pid))
	}
	if r, _, _ := messageBoxW.Call(uintptr(hwndMain), uintptr(unsafe.Pointer(u16p(strings.Join(lines, "\n")))), uintptr(unsafe.Pointer(u16p("确认结束进程"))), MB_YESNO|MB_ICONWARNING|MB_DEFBUTTON2); r != 6 {
		return
	}
	startKill(ts, fmt.Sprintf("端口 %d", port))
}

// ---------------- 绘制（双缓冲，微信风格） ----------------

const (
	sideW   = 68
	padL    = 28
	rowH    = 64
	cardTop = 124
	cardBot = 40
)

func contentH() int { return int(gcl.b) - cardTop - cardBot }

func paintAll(hdc syscall.Handle) {
	var r rectT
	getClientRect.Call(uintptr(hwndMain), uintptr(unsafe.Pointer(&r)))
	gcl = r
	W, H := int(r.r), int(r.b)
	if W <= 0 || H <= 0 {
		return
	}

	mem, _, _ := createCompatibleDC.Call(uintptr(hdc))
	bmp, _, _ := createCompatibleBitmap.Call(uintptr(hdc), uintptr(W), uintptr(H))
	oldBmp, _, _ := selectObject.Call(mem, bmp)
	defer func() {
		selectObject.Call(mem, oldBmp)
		deleteObject.Call(bmp)
		deleteDC.Call(mem)
	}()

	hits = hits[:0]
	m := syscall.Handle(mem)
	setBkMode.Call(mem, TRANSPARENT)

	// 页面背景 + 左侧深色栏
	fill(m, 0, 0, W, H, cPage)
	fill(m, 0, 0, sideW, H, cSide)
	lxc, lyc := sideW/2, 46
	roundRect(m, lxc-20, lyc-20, lxc+20, lyc+20, 40, 40, cGreen, 0)
	text(m, "端", lxc-20, lyc-20, lxc+20, lyc+20, fLogo, cCard, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
	roundRect(m, sideW/2-4, 86, sideW/2+4, 94, 8, 8, cref(0x66, 0x66, 0x66), 0)
	roundRect(m, sideW/2-4, 104, sideW/2+4, 112, 8, 8, cref(0x44, 0x44, 0x44), 0)

	x0 := sideW + padL

	// 标题 + 副标题
	text(m, "端口管理", x0, 20, x0+300, 52, fH1, cText, DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
	sub := fmt.Sprintf("%d 个监听端口 · 更新于 %s", len(allData), lastTime)
	if lastTime == "" {
		sub = "正在加载…"
	}
	text(m, sub, x0, 54, x0+420, 74, fSm, cSub, DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	// 搜索框（右上胶囊）
	sbw := 300
	sx, sy := W-padL-sbw, 26
	sr := rectT{int32(sx), int32(sy), int32(sx + sbw), int32(sy + 38)}
	hits = append(hits, hitZone{kind: hitSearch, r: sr})
	roundRect(m, int(sr.l), int(sr.t), int(sr.r), int(sr.b), 19, 19, cCard, boolRef(focused, cGreen, cBorder))
	icx, icy := sx+16, sy+14
	roundRect(m, icx, icy-6, icx+12, icy+6, 12, 12, cCard, cSub)
	drawLine(m, icx+10, icy+7, icx+16, icy+13, cSub)
	qStr := string(q)
	if qStr == "" {
		text(m, "搜索端口 / 进程名", sx+38, sy, sx+sbw-40, sy+38, fTx, cPlaceholder, DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
	} else {
		text(m, qStr, sx+38, sy, sx+sbw-56, sy+38, fTx, cText, DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_END_ELLIPSIS|DT_NOPREFIX)
		if focused && caretOn {
			tw := measure(m, fTx, qStr)
			fill(m, sx+40+tw, sy+9, sx+42+tw, sy+29, cGreen)
		}
		ccx, ccy := sx+sbw-24, sy+19
		cr := rectT{int32(ccx - 10), int32(ccy - 10), int32(ccx + 10), int32(ccy + 10)}
		hits = append(hits, hitZone{kind: hitClear, r: cr})
		roundRect(m, int(cr.l), int(cr.t), int(cr.r), int(cr.b), 20, 20,
			boolRef(hover.kind == hitClear, cref(0xDA, 0xDA, 0xDA), cref(0xEE, 0xEE, 0xEE)), 0)
		drawLine(m, ccx-4, ccy-4, ccx+4, ccy+4, cref(0x88, 0x88, 0x88))
		drawLine(m, ccx+4, ccy-4, ccx-4, ccy+4, cref(0x88, 0x88, 0x88))
	}

	// 工具栏
	ty := 84
	bw := 84
	br := rectT{int32(x0), int32(ty), int32(x0 + bw), int32(ty + 30)}
	hits = append(hits, hitZone{kind: hitBtnRefresh, r: br})
	roundRect(m, int(br.l), int(br.t), int(br.r), int(br.b), 15, 15,
		boolRef(hover.kind == hitBtnRefresh, cGrayHov, cCard), cBorder)
	text(m, "刷新", int(br.l), int(br.t), int(br.r), int(br.b), fTx, cText, DT_CENTER|DT_VCENTER|DT_SINGLELINE)

	kbw := 176
	kr := rectT{int32(x0 + bw + 12), int32(ty), int32(x0 + bw + 12 + kbw), int32(ty + 30)}
	hits = append(hits, hitZone{kind: hitBtnKillPort, r: kr})
	enabled := !killBusy && selected >= 0 && selected < len(shown)
	kb := cGreenDis
	if enabled {
		kb = boolRef(hover.kind == hitBtnKillPort, cGreenHov, cGreen)
	}
	roundRect(m, int(kr.l), int(kr.t), int(kr.r), int(kr.b), 15, 15, kb, 0)
	text(m, "结束该端口全部进程", int(kr.l), int(kr.t), int(kr.r), int(kr.b), fTx, cCard, DT_CENTER|DT_VCENTER|DT_SINGLELINE)

	// 自动刷新开关
	tw2 := 44
	sx2 := W - padL - tw2
	tr := rectT{int32(sx2), int32(ty + 3), int32(sx2 + tw2), int32(ty + 25)}
	hits = append(hits, hitZone{kind: hitToggle, r: tr})
	roundRect(m, int(tr.l), int(tr.t), int(tr.r), int(tr.b), 22, 22, boolRef(autoOn, cGreen, cSwitchOff), 0)
	kx := int(tr.l) + 3
	if autoOn {
		kx = int(tr.r) - 21
	}
	knob(m, kx, int(tr.t)+2, 18, cCard)
	text(m, "自动刷新", int(tr.l)-76, int(tr.t)-3, int(tr.l)-6, int(tr.b)+3, fSm, cSub, DT_RIGHT|DT_VCENTER|DT_SINGLELINE)

	// 列表卡片
	cx1, cx2 := x0, W-padL
	cy1, cy2 := cardTop, H-cardBot
	roundRect(m, cx1, cy1, cx2, cy2, 10, 10, cCard, cLine)

	if len(shown) == 0 {
		empty := "没有匹配的监听端口" + boolStr(qStr != "", "，试试清空搜索框", "，稍后自动重试")
		text(m, empty, cx1, cy1, cx2, cy2, fTx, cSub, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
	}
	for i := range shown {
		ry0 := cy1 + i*rowH - scrollY
		ry1 := ry0 + rowH
		if ry1 <= cy1+3 || ry0 >= cy2-3 {
			continue
		}
		c := shown[i]
		rowR := rectT{int32(cx1) + 1, int32(ry0), int32(cx2) - 1, int32(ry1)}
		hits = append(hits, hitZone{kind: hitRow, idx: i, r: rowR})
		if selected == i {
			fill(m, int(rowR.l), int(rowR.t), int(rowR.r), int(rowR.b), cGreenBg)
			fill(m, int(rowR.l), int(rowR.t), int(rowR.l)+4, int(rowR.b), cGreen)
		} else if hover.kind == hitRow && hover.idx == i {
			fill(m, int(rowR.l), int(rowR.t), int(rowR.r), int(rowR.b), cHover)
		}
		udp := c.Proto == "UDP"
		bcol, bbg := cGreen, cGreenBg
		if udp {
			bcol, bbg = cOrange, cOrangeBg
		}
		bx1 := int(rowR.l) + 16
		roundRect(m, bx1, ry0+21, bx1+46, ry0+43, 6, 6, bbg, 0)
		text(m, c.Proto, bx1, ry0+21, bx1+46, ry0+43, fSm, bcol, DT_CENTER|DT_VCENTER|DT_SINGLELINE)

		text(m, fmt.Sprintf("%d", c.Port), bx1+60, ry0, bx1+60+96, ry0+rowH, fBig, cText, DT_LEFT|DT_VCENTER|DT_SINGLELINE)

		tx := bx1 + 162
		name := c.Name
		if name == "?" || name == "" {
			name = "(未知进程)"
		}
		text(m, name, tx, ry0+8, int(rowR.r)-130, ry0+30, fBd, cText, DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_END_ELLIPSIS)
		text(m, fmt.Sprintf("PID %d · %s", c.PID, c.Addr), tx, ry0+32, int(rowR.r)-130, ry0+54, fSm, cSub, DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_END_ELLIPSIS)

		kx1, kx2 := int(rowR.r)-116, int(rowR.r)-20
		kyr := rectT{int32(kx1), int32(ry0 + 16), int32(kx2), int32(ry0 + 46)}
		hits = append(hits, hitZone{kind: hitRowKill, idx: i, r: kyr})
		hov := hover.kind == hitRowKill && hover.idx == i
		bg := boolRef(hov, cRedBg, cCard)
		if killBusy {
			bg = cHover
		}
		roundRect(m, kx1, ry0+16, kx2, ry0+46, 6, 6, bg, boolRef(hov, cRed, cRedBorder))
		text(m, "结束进程", kx1, ry0+16, kx2, ry0+46, fTx, boolRef(killBusy, cSub, cRed), DT_CENTER|DT_VCENTER|DT_SINGLELINE)

		if i < len(shown)-1 {
			drawLine(m, int(rowR.l)+14, ry1, int(rowR.r)-14, ry1, cLine)
		}
	}

	// 滚动条
	total := len(shown) * rowH
	chh := contentH()
	if total > chh && chh > 0 {
		trackH := cy2 - cy1 - 8
		thumbH := int(float64(chh) / float64(total) * float64(trackH))
		if thumbH < 30 {
			thumbH = 30
		}
		if thumbH > trackH {
			thumbH = trackH
		}
		ty2 := cy1 + 4 + int(float64(scrollY)/float64(total-chh)*float64(trackH-thumbH))
		roundRect(m, cx2-11, ty2, cx2-5, ty2+thumbH, 6, 6, cScrollBar, 0)
	}

	// 状态栏
	st := fmt.Sprintf("显示 %d / %d 条", len(shown), len(allData))
	if statusMsg != "" {
		st = statusMsg
	}
	text(m, st, x0, H-30, W/2, H-8, fSm, cSub, DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_END_ELLIPSIS)
	a, _, _ := isUserAdmin.Call()
	hint := "以管理员身份运行可结束系统进程"
	if a != 0 {
		hint = "管理员权限已就绪"
	}
	text(m, hint, W/2, H-30, W-padL, H-8, fSm, boolRef(a != 0, cGreen, cSub), DT_RIGHT|DT_VCENTER|DT_SINGLELINE)

	bitBlt.Call(uintptr(hdc), 0, 0, uintptr(W), uintptr(H), mem, 0, 0, SRCCOPY)
}

// ---------------- 绘图助手 ----------------

func boolRef(cond bool, a, b uintptr) uintptr {
	if cond {
		return a
	}
	return b
}

func boolStr(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

func fill(m syscall.Handle, x1, y1, x2, y2 int, c uintptr) {
	r := rectT{int32(x1), int32(y1), int32(x2), int32(y2)}
	b, _, _ := createSolidBrush.Call(c)
	fillRect.Call(uintptr(m), uintptr(unsafe.Pointer(&r)), b)
	deleteObject.Call(b)
}

func roundRect(m syscall.Handle, x1, y1, x2, y2, rx, ry int, fillC, penC uintptr) {
	var pen uintptr
	if penC != 0 {
		p, _, _ := createPen.Call(PS_SOLID, 1, penC)
		pen = p
	} else {
		np, _, _ := getStockObject.Call(NULL_PEN)
		pen = np
	}
	b, _, _ := createSolidBrush.Call(fillC)
	oldP, _, _ := selectObject.Call(uintptr(m), pen)
	oldB, _, _ := selectObject.Call(uintptr(m), b)
	roundRectProc.Call(uintptr(m), uintptr(x1), uintptr(y1), uintptr(x2), uintptr(y2), uintptr(rx), uintptr(ry))
	selectObject.Call(uintptr(m), oldP)
	selectObject.Call(uintptr(m), oldB)
	deleteObject.Call(b)
	if penC != 0 {
		deleteObject.Call(pen)
	}
}

func knob(m syscall.Handle, x, y, d int, c uintptr) {
	b, _, _ := createSolidBrush.Call(c)
	np, _, _ := getStockObject.Call(NULL_PEN)
	oldB, _, _ := selectObject.Call(uintptr(m), b)
	oldP, _, _ := selectObject.Call(uintptr(m), np)
	ellipseProc.Call(uintptr(m), uintptr(x), uintptr(y), uintptr(x+d), uintptr(y+d))
	selectObject.Call(uintptr(m), oldB)
	selectObject.Call(uintptr(m), oldP)
	deleteObject.Call(b)
}

func drawLine(m syscall.Handle, x1, y1, x2, y2 int, c uintptr) {
	pen, _, _ := createPen.Call(PS_SOLID, 2, c)
	old, _, _ := selectObject.Call(uintptr(m), pen)
	moveToEx.Call(uintptr(m), uintptr(x1), uintptr(y1), 0)
	lineTo.Call(uintptr(m), uintptr(x2), uintptr(y2))
	selectObject.Call(uintptr(m), old)
	deleteObject.Call(pen)
}

func text(m syscall.Handle, s string, x1, y1, x2, y2 int, f syscall.Handle, c uintptr, flags uint32) {
	setTextColor.Call(uintptr(m), c)
	selectObject.Call(uintptr(m), uintptr(f))
	r := rectT{int32(x1), int32(y1), int32(x2), int32(y2)}
	p := u16p(s)
	drawTextW.Call(uintptr(m), uintptr(unsafe.Pointer(p)), uintptr(u16len(s)),
		uintptr(unsafe.Pointer(&r)), uintptr(flags))
}

func measure(m syscall.Handle, f syscall.Handle, s string) int {
	selectObject.Call(uintptr(m), uintptr(f))
	var sz sizeT
	p := u16p(s)
	getTextExtentPoint32W.Call(uintptr(m), uintptr(unsafe.Pointer(p)), uintptr(u16len(s)), uintptr(unsafe.Pointer(&sz)))
	return int(sz.cx)
}
