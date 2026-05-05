// Package ping implements ICMP echo request/reply (ping) without
// relying on the system ping command. It constructs and sends raw
// ICMP packets and parses responses.
package ping

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"time"
)

// ICMP type constants
const (
	TypeEchoReply   = 0
	TypeEchoRequest = 8
)

// ICMP header (8 bytes)
type icmpHeader struct {
	Type     uint8
	Code     uint8
	Checksum uint16
	ID       uint16
	Seq      uint16
}

// Result holds the outcome of a single ICMP echo request.
type Result struct {
	IP      string
	Success bool
	RTT     time.Duration
	Size    int // payload bytes received
	TTL     int
	Err     error
}

// Ping sends one ICMP echo request to the target ip address and waits up
// to timeout for a reply. Privileged (root / CAP_NET_RAW) access is required
// to open the raw ICMP socket.
func Ping(ip string, timeout time.Duration) Result {
	start := time.Now()

	raddr, err := net.ResolveIPAddr("ip", ip)
	if err != nil {
		return Result{IP: ip, Err: fmt.Errorf("resolve: %w", err)}
	}

	conn, err := net.DialIP("ip4:icmp", nil, raddr)
	if err != nil {
		return Result{IP: ip, Err: fmt.Errorf("socket: %w", err)}
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(timeout))

	id := uint16(os.Getpid() & 0xFFFF)
	seq := uint16(1)

	pkt := buildEchoRequest(id, seq, start)

	if _, err := conn.Write(pkt); err != nil {
		return Result{IP: ip, Err: fmt.Errorf("send: %w", err)}
	}

	// Read in a loop to consume responses that don't match (e.g. from other
	// processes). Time out after approximately timeout.
	var respRslt Result
	for {
		buf := make([]byte, 1500)
		n, err := conn.Read(buf)
		if err != nil {
			return Result{IP: ip, Err: fmt.Errorf("recv: %w", err)}
		}

		// net.DialIP on darwin includes the IP header in the read buffer,
		// while on Linux it is stripped. Detect and skip accordingly.
		off := 0
		if len(buf) > 0 && (buf[0]>>4)&0x0F == 4 {
			off = int(buf[0]&0x0F) * 4 // IHL in 32-bit words
		} else if len(buf) > 0 && (buf[0]>>4)&0x0F == 6 {
			off = 40 // IPv6 fixed header
		}
		if n < off+8 {
			continue
		}
		icmpStart := off

		var hdr icmpHeader
		if err := binary.Read(bytes.NewReader(buf[icmpStart:]), binary.BigEndian, &hdr); err != nil {
			continue
		}

		// We only care about Echo Reply destined for our ID.
		if hdr.Type != TypeEchoReply || hdr.ID != id {
			continue
		}

		// Extract TTL from IP header if available.
		ttl := 0
		if off >= 9 {
			ttl = int(buf[8])
		}

		respRslt = Result{
			IP:      ip,
			Success: true,
			RTT:     time.Since(start),
			Size:    n - icmpStart,
			TTL:     ttl,
		}
		break
	}
	return respRslt
}

// buildEchoRequest creates a serialised ICMP echo request with a timestamp
// payload and computes the correct checksum.
func buildEchoRequest(id, seq uint16, ts time.Time) []byte {
	hdr := icmpHeader{
		Type: TypeEchoRequest,
		Code: 0,
		ID:   id,
		Seq:  seq,
	}

	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.BigEndian, hdr)

	// Timestamp payload used to reconstruct RTT on the receiving side.
	payload := make([]byte, 8)
	binary.BigEndian.PutUint64(payload, uint64(ts.UnixNano()))
	buf.Write(payload)

	pkt := buf.Bytes()

	// Compute RFC 1071 checksum over the ICMP segment.
	csum := ipChecksum(pkt)
	pkt[2] = byte(csum >> 8)
	pkt[3] = byte(csum & 0xFF)

	return pkt
}

// ipChecksum computes the 16-bit one's complement checksum (RFC 1071).
func ipChecksum(data []byte) uint16 {
	var sum uint32
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(data[i])<<8 | uint32(data[i+1])
	}
	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}
	for (sum >> 16) > 0 {
		sum = (sum >> 16) + (sum & 0xFFFF)
	}
	return ^uint16(sum)
}

// ParsePacket extracts the ICMP header and payload from raw bytes. The caller
// is responsible for stripping any preceding IP header.
func ParsePacket(data []byte) (typeByte uint8, code uint8, id uint16, seq uint16, payload []byte, err error) {
	if len(data) < 8 {
		err = fmt.Errorf("packet too short: %d bytes", len(data))
		return
	}
	hdr := icmpHeader{}
	if err = binary.Read(bytes.NewReader(data[:8]), binary.BigEndian, &hdr); err != nil {
		return
	}
	return hdr.Type, hdr.Code, hdr.ID, hdr.Seq, data[8:], nil
}

// ValidateChecksum returns true when the ICMP packet's checksum is correct.
// data must begin at the ICMP header (no IP header prefix).
func ValidateChecksum(data []byte) bool {
	if len(data) < 8 {
		return false
	}
	saved := binary.BigEndian.Uint16(data[2:4])
	data[2], data[3] = 0, 0
	got := ipChecksum(data)
	data[2], data[3] = byte(saved>>8), byte(saved&0xFF)
	return got == saved
}
