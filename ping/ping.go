package ping

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Result 表示ping操作的结果
type Result struct {
	Target   string
	Success  bool
	Sent     int
	Received int
	Lost     int
	LossRate float64
	Output   string
	Error    string
}

// Ping 对指定IP地址执行ping操作
// ip: 目标IP地址
// count: 发送的ICMP包数量（默认4个）
// timeout: 单包超时时间（默认5秒）
func Ping(ip string, count int, timeout time.Duration) (*Result, error) {
	if count <= 0 {
		count = 4
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	args := buildPingArgs(ip, count, timeout)
	cmd := exec.Command("ping", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := strings.TrimSpace(stdout.String())
	errStr := strings.TrimSpace(stderr.String())

	result := &Result{
		Target: ip,
		Sent:   count,
		Output: output,
	}

	if err != nil {
		result.Error = fmt.Sprintf("ping failed: %v", err)
		if errStr != "" {
			result.Error += ", stderr: " + errStr
		}
		return result, nil
	}

	parseOutput(result, output)
	return result, nil
}

// QuickPing 对指定IP快速执行一次ping（4个包，5秒超时）
func QuickPing(ip string) (*Result, error) {
	return Ping(ip, 4, 5*time.Second)
}

// buildPingArgs 根据操作系统构建ping命令参数
func buildPingArgs(ip string, count int, timeout time.Duration) []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{"-c", strconv.Itoa(count), "-t", strconv.Itoa(int(timeout.Seconds())), ip}
	case "linux":
		return []string{"-c", strconv.Itoa(count), "-W", strconv.Itoa(int(timeout.Seconds())), ip}
	default:
		return []string{"-n", strconv.Itoa(count), "-w", strconv.Itoa(int(timeout.Milliseconds())), ip}
	}
}

// parseOutput 解析ping命令的标准输出
func parseOutput(result *Result, output string) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 统计收到回复的行
		if strings.Contains(line, "bytes from") || strings.Contains(line, "time=") ||
			strings.Contains(line, "time<") {
			result.Received++
		}
	}
	result.Lost = result.Sent - result.Received
	if result.Sent > 0 {
		result.LossRate = float64(result.Lost) / float64(result.Sent) * 100
	}
	result.Success = result.Received > 0
}
