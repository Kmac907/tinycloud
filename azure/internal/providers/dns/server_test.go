package dns

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"

	"tinycloud/internal/config"
	"tinycloud/internal/state"
	"tinycloud/internal/telemetry"
)

func TestHandleQueryReturnsARecordAnswers(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, err := store.UpsertPrivateDNSZone("sub-123", "rg-one", "internal.test", nil); err != nil {
		t.Fatalf("UpsertPrivateDNSZone() error = %v", err)
	}
	if _, err := store.UpsertPrivateDNSARecordSet("sub-123", "rg-one", "internal.test", "api", 60, []string{"10.0.0.4", "10.0.0.5"}); err != nil {
		t.Fatalf("UpsertPrivateDNSARecordSet() error = %v", err)
	}

	server := NewServer(store, config.FromEnv(), telemetry.NewJSONLogger(bytes.NewBuffer(nil)))
	response := server.handleQuery(buildQuery("api.internal.test", dnsTypeA))

	if rcode(response) != dnsRCodeSuccess {
		t.Fatalf("rcode = %d, want %d", rcode(response), dnsRCodeSuccess)
	}
	if answerCount(response) != 2 {
		t.Fatalf("answerCount = %d, want %d", answerCount(response), 2)
	}
	addresses := answerAddresses(response)
	if len(addresses) != 2 || addresses[0] != "10.0.0.4" || addresses[1] != "10.0.0.5" {
		t.Fatalf("addresses = %#v, want %#v", addresses, []string{"10.0.0.4", "10.0.0.5"})
	}
}

func TestHandleQueryReturnsNXDomainForMissingRecord(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	server := NewServer(store, config.FromEnv(), telemetry.NewJSONLogger(bytes.NewBuffer(nil)))
	response := server.handleQuery(buildQuery("missing.internal.test", dnsTypeA))

	if rcode(response) != dnsRCodeNameError {
		t.Fatalf("rcode = %d, want %d", rcode(response), dnsRCodeNameError)
	}
	if answerCount(response) != 0 {
		t.Fatalf("answerCount = %d, want %d", answerCount(response), 0)
	}
}

func buildQuery(name string, qtype uint16) []byte {
	packet := make([]byte, 12)
	binary.BigEndian.PutUint16(packet[0:2], 0x1234)
	binary.BigEndian.PutUint16(packet[2:4], dnsFlagRD)
	binary.BigEndian.PutUint16(packet[4:6], 1)

	for _, label := range bytes.Split([]byte(name), []byte(".")) {
		packet = append(packet, byte(len(label)))
		packet = append(packet, label...)
	}
	packet = append(packet, 0)
	packet = binary.BigEndian.AppendUint16(packet, qtype)
	packet = binary.BigEndian.AppendUint16(packet, dnsClassIN)
	return packet
}

func rcode(packet []byte) uint16 {
	return binary.BigEndian.Uint16(packet[2:4]) & 0x000f
}

func answerCount(packet []byte) uint16 {
	return binary.BigEndian.Uint16(packet[6:8])
}

func answerAddresses(packet []byte) []string {
	offset := 12
	for {
		length := int(packet[offset])
		offset++
		if length == 0 {
			break
		}
		offset += length
	}
	offset += 4

	count := int(answerCount(packet))
	addresses := make([]string, 0, count)
	for i := 0; i < count; i++ {
		offset += 2
		offset += 2
		offset += 2
		offset += 4
		rdLength := int(binary.BigEndian.Uint16(packet[offset : offset+2]))
		offset += 2
		addresses = append(addresses, net.IP(packet[offset:offset+rdLength]).String())
		offset += rdLength
	}
	return addresses
}
