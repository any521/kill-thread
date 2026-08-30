package main

import "syscall"

// hideWindowAttr 阻止 Windows 为 netstat/tasklist/taskkill 弹出控制台黑框
func hideWindowAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true}
}
