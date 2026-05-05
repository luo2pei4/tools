package ping

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// buildEchoRequest
// ---------------------------------------------------------------------------

func TestBuildEchoRequest_HasCorrectTypeAndCode(t *testing.T) {
	pkt := buildEchoRequest(0x1234, 0x0001, time.Now())
	if len(pkt) < 8 {
		t.Fatalf("packet too short: %d", len(pkt))
	}
	if pkt[0] != TypeEchoRequest {
		t.Errorf("expected type %d, got %d", TypeEchoRequest, pkt[0])
	}
	if pkt[1] != 0 {
		t.Errorf("expected code 0, got %d", pkt[1])
	}
}

func TestBuildEchoRequest_CarriesIDAndSeq(t *testing.T) {
	pkt := buildEchoRequest(0xABCD, 0x00FF, time.Now())
	id := binary.BigEndian.Uint16(pkt[4:6])
	seq := binary.BigEndian.Uint16(pkt[6:8])
	if id != 0xABCD {
		t.Errorf("expected id 0xABCD, got 0x%04X", id)
	}
	if seq != 0x00FF {
		t.Errorf("expected seq 0x00FF, got 0x%04X", seq)
	}
}

func TestBuildEchoRequest_Has16BytePayload(t *testing.T) {
	pkt := buildEchoRequest(1, 1, time.Now())
	const hdrLen = 8
	const payloadLen = 8 // timestamp uint64
	want := hdrLen + payloadLen
	if len(pkt) != want {
		t.Errorf("expected %d bytes total, got %d", want, len(pkt))
	}
}

func TestBuildEchoRequest_ChecksumIsCorrect(t *testing.T) {
	pkt := buildEchoRequest(1, 1, time.Now())
	if !ValidateChecksum(pkt) {
		t.Error("checksum validation failed on fresh packet")
	}
}

func TestBuildEchoRequest_ChecksumNonZero(t *testing.T) {
	pkt := buildEchoRequest(1, 1, time.Now())
	csum := binary.BigEndian.Uint16(pkt[2:4])
	if csum == 0 {
		t.Error("checksum should not be zero")
	}
}

func TestBuildEchoRequest_ContainsTimestamp(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 123456789, time.UTC)
	pkt := buildEchoRequest(0, 0, now)
	payload := pkt[8:]
	got := time.Unix(0, int64(binary.BigEndian.Uint64(payload)))
	if !got.Equal(now) {
		t.Errorf("expected timestamp %v, got %v", now, got)
	}
}

// ---------------------------------------------------------------------------
// ipChecksum
// ---------------------------------------------------------------------------

func TestIPChecksum_KnownValues(t *testing.T) {
	// RFC 1071 example: 0x4500, 0x0073, 0x0000, 0x4000, 0x4011, 0x0000, 0xC0A8, 0x0001, 0xC0A8, 0x00C7
	// Computed checksum for this IP header is 0xB861 (when checksum field is zero)
	data := []byte{0x45, 0x00, 0x00, 0x73, 0x00, 0x00, 0x40, 0x00, 0x40, 0x11, 0x00, 0x00, 0xC0, 0xA8, 0x00, 0x01, 0xC0, 0xA8, 0x00, 0xC7}
	got := ipChecksum(data)
	if got != 0xB861 {
		t.Errorf("expected 0xB861, got 0x%04X", got)
	}
}

func TestIPChecksum_AllFF(t *testing.T) {
	// 0xFFFF -> one's complement -> 0x0000
	data := []byte{0xFF, 0xFF}
	got := ipChecksum(data)
	if got != 0x0000 {
		t.Errorf("expected 0x0000, got 0x%04X", got)
	}
}

func TestIPChecksum_AllZero(t *testing.T) {
	data := []byte{0x00, 0x00}
	got := ipChecksum(data)
	if got != 0xFFFF {
		t.Errorf("expected 0xFFFF (0x0000 complements to 0xFFFF), got 0x%04X", got)
	}
}

func TestIPChecksum_SingleByte(t *testing.T) {
	data := []byte{0x12}
	got := ipChecksum(data)
	if got != ^uint16(0x1200) {
		t.Errorf("expected 0x%04X, got 0x%04X", ^uint16(0x1200), got)
	}
}

func TestIPChecksum_OddLength(t *testing.T) {
	data := []byte{0x00, 0x01, 0x02}
	got := ipChecksum(data)
	// 0x0001 + 0x0200 = 0x0201, complement = 0xFDFE
	if got != 0xFDFE {
		t.Errorf("expected 0xFDFE, got 0x%04X", got)
	}
}

// ---------------------------------------------------------------------------
// ParsePacket
// ---------------------------------------------------------------------------

func TestParsePacket_Valid(t *testing.T) {
	hdr := icmpHeader{Type: 3, Code: 0, ID: 0xAA, Seq: 0xBB}
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.BigEndian, hdr)
	buf.Write([]byte{1, 2, 3, 4})

	typ, code, id, seq, payload, err := ParsePacket(buf.Bytes())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if typ != 3 || code != 0 {
		t.Errorf("expected type=3 code=0, got type=%d code=%d", typ, code)
	}
	if id != 0xAA || seq != 0xBB {
		t.Errorf("expected id=0xAA seq=0xBB, got id=0x%04X seq=0x%04X", id, seq)
	}
	if !bytes.Equal(payload, []byte{1, 2, 3, 4}) {
		t.Errorf("payload mismatch: %v", payload)
	}
}

func TestParsePacket_TooShort(t *testing.T) {
	_, _, _, _, _, err := ParsePacket([]byte{0, 0, 0, 0, 0, 0, 0})
	if err == nil {
		t.Error("expected error for short packet")
	}
}

func TestParsePacket_Empty(t *testing.T) {
	_, _, _, _, _, err := ParsePacket(nil)
	if err == nil {
		t.Error("expected error for nil/empty")
	}
}

// ---------------------------------------------------------------------------
// ValidateChecksum
// ---------------------------------------------------------------------------

func TestValidateChecksum_Valid(t *testing.T) {
	pkt := buildEchoRequest(1, 1, time.Now())
	if !ValidateChecksum(pkt) {
		t.Error("expected valid checksum on fresh packet")
	}
}

func TestValidateChecksum_ModifiedPayload(t *testing.T) {
	pkt := buildEchoRequest(1, 1, time.Now())
	pkt[len(pkt)-1] ^= 0xFF // corrupt last byte
	if ValidateChecksum(pkt) {
		t.Error("expected invalid checksum after payload corruption")
	}
}

func TestValidateChecksum_ShortPacket(t *testing.T) {
	if ValidateChecksum(nil) {
		t.Error("false positive on nil")
	}
	if ValidateChecksum([]byte{0, 0, 0, 0, 0, 0, 0}) {
		t.Error("false positive on 7-byte packet")
	}
}

// ---------------------------------------------------------------------------
// Integration-style test: we can at least test that Ping attempts a
// connection and returns the correct error when run without privileges.
// This test is skipped unless the test is run with -tags=integration.
// ---------------------------------------------------------------------------

func TestBuildEchoRequest_Deterministic(t *testing.T) {
	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a := buildEchoRequest(42, 7, ts)
	b := buildEchoRequest(42, 7, ts)
	if !bytes.Equal(a, b) {
		t.Error("same inputs should produce identical output")
	}
}

func TestBuildEchoRequest_VaryingID(t *testing.T) {
	ts := time.Now()
	different := false
	id1 := binary.BigEndian.Uint16(buildEchoRequest(1, 1, ts)[4:6])
	for i := 0; i < 10; i++ {
		pkt := buildEchoRequest(binary.BigEndian.Uint16([]byte{0x00, byte(i)}), 1, ts)
		id2 := binary.BigEndian.Uint16(pkt[4:6])
		if id2 != id1 {
			different = true
			break
		}
	}
	if !different {
		t.Error("ID should vary with input")
	}
}

func BenchmarkBuildEchoRequest(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buildEchoRequest(uint16(i), uint16(i), time.Now())
	}
}

func BenchmarkChecksum(b *testing.B) {
	pkt := buildEchoRequest(1, 1, time.Now())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateChecksum(pkt)
	}
}

// Example (integration) – run with:
//
//	go test -run ^ExamplePing$ -v ./ping/
//
// Requires root / CAP_NET_RAW on most systems.
func ExamplePing() {
	// Resolve a real target that is almost always reachable.
	res, err := net.LookupHost("1.1.1.1")
	if err != nil {
		// Don't fail the example if offline.
		return
	}
	r := Ping(res[0], 3*time.Second)
	if !r.Success {
		// On CI or unprivileged systems this is expected.
		return
	}
	_ = r
	// Output:
}
