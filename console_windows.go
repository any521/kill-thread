//go:build windows

package main

import "syscall"

// setupConsole 把 Windows 控制台输入/输出代码页切换为 UTF-8，避免中文乱码。
func setupConsole() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	if p := kernel32.NewProc("SetConsoleOutputCP"); p.Find() == nil {
		p.Call(65001) // CP_UTF8
	}
	if p := kernel32.NewProc("SetConsoleCP"); p.Find() == nil {
		p.Call(65001)
	}
}
