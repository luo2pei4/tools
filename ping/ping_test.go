package ping

import (
	"fmt"
	"testing"
	"time"
)

// 测试解析函数
func TestParseOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		sent     int
		wantRecv int
		wantSucc bool
	}{
		{
			name: "macOS 全部成功",
			output: `PING 8.8.8.8 (8.8.8.8): 56 data bytes
64 bytes from 8.8.8.8: icmp_seq=0 ttl=117 time=11.432 ms
64 bytes from 8.8.8.8: icmp_seq=1 ttl=117 time=11.211 ms
64 bytes from 8.8.8.8: icmp_seq=2 ttl=117 time=11.089 ms
64 bytes from 8.8.8.8: icmp_seq=3 ttl=117 time=11.345 ms

--- 8.8.8.8 ping statistics ---
4 packets transmitted, 4 packets received, 0.0% packet loss
round-trip min/avg/max/stddev = 11.089/11.269/11.432/0.134 ms`,
			sent:     4,
			wantRecv: 4,
			wantSucc: true,
		},
		{
			name: "macOS 部分丢包",
			output: `PING 192.168.1.100 (192.168.1.100): 56 data bytes
64 bytes from 192.168.1.100: icmp_seq=0 ttl=64 time=1.234 ms
64 bytes from 192.168.1.100: icmp_seq=1 ttl=64 time=1.567 ms
--- 192.168.1.100 ping statistics ---
4 packets transmitted, 2 packets received, 50.0% packet loss
round-trip min/avg/max/stddev = 1.234/1.400/1.567/0.167 ms`,
			sent:     4,
			wantRecv: 2,
			wantSucc: true,
		},
		{
			name: "Linux 输出格式",
			output: `PING 8.8.8.8 (8.8.8.8) 56(84) bytes of data.
64 bytes from 8.8.8.8: icmp_seq=1 ttl=117 time=11.4 ms
64 bytes from 8.8.8.8: icmp_seq=2 ttl=117 time=11.3 ms

--- 8.8.8.8 ping statistics ---
2 packets transmitted, 2 received, 0% packet loss, time 1001ms
rtt min/avg/max/mdev = 11.300/11.350/11.400/0.050 ms`,
			sent:     2,
			wantRecv: 2,
			wantSucc: true,
		},
		{
			name: "Windows 输出格式",
			output: `Pinging 8.8.8.8 with 32 bytes of data:
Reply from 8.8.8.8: bytes=32 time=11ms TTL=117
Reply from 8.8.8.8: bytes=32 time=12ms TTL=117
Reply from 8.8.8.8: bytes=32 time=10ms TTL=117
Reply from 8.8.8.8: bytes=32 time=11ms TTL=117

Ping statistics for 8.8.8.8:
    Packets: Sent = 4, Received = 4, Lost = 0 (0% loss)`,
			sent:     4,
			wantRecv: 4,
			wantSucc: true,
		},
		{
			name: "全部丢包",
			output: `PING 10.255.255.1 (10.255.255.1): 56 data bytes

--- 10.255.255.1 ping statistics ---
3 packets transmitted, 0 packets received, 100.0% packet loss`,
			sent:     3,
			wantRecv: 0,
			wantSucc: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &Result{
				Target: "test",
				Sent:   tt.sent,
			}
			parseOutput(result, tt.output)
			if result.Received != tt.wantRecv {
				t.Errorf("parseOutput() got Received=%d, want %d", result.Received, tt.wantRecv)
			}
			if result.Success != tt.wantSucc {
				t.Errorf("parseOutput() got Success=%v, want %v", result.Success, tt.wantSucc)
			}
		})
	}
}

// 测试buildPingArgs函数
func TestBuildPingArgs(t *testing.T) {
	args := buildPingArgs("8.8.8.8", 4, 5*time.Second)
	if len(args) == 0 {
		t.Error("buildPingArgs() returned empty args")
	}
	if args[len(args)-1] != "8.8.8.8" {
		t.Errorf("buildPingArgs() last arg should be IP, got %s", args[len(args)-1])
	}
}

// TestQuickPing_Localhost 对127.0.0.1执行ping测试（需要网络栈支持）
func TestQuickPing_Localhost(t *testing.T) {
	result, err := QuickPing("127.0.0.1")
	if err != nil {
		t.Fatalf("QuickPing failed: %v", err)
	}
	if !result.Success {
		t.Errorf("ping 127.0.0.1 should succeed, got: %s", result.Error)
	}
	fmt.Printf("localhost ping result: Sent=%d, Received=%d, LossRate=%.1f%%\n",
		result.Sent, result.Received, result.LossRate)
}

// TestPing_InvalidIP 对无效IP执行ping测试
func TestPing_InvalidIP(t *testing.T) {
	result, err := Ping("999.999.999.999", 2, 2*time.Second)
	if err != nil {
		t.Fatalf("Ping() unexpected error: %v", err)
	}
	// 无效IP可能返回错误也可能不返回，取决于系统实现
	// 只要不panic就算通过
	fmt.Printf("invalid IP ping result: Success=%v, Error=%s\n", result.Success, result.Error)
}
