package dns

import (
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"

	"tinycloud/internal/config"
	"tinycloud/internal/state"
	"tinycloud/internal/telemetry"
)

const (
	dnsTypeA      = 1
	dnsClassIN    = 1
	dnsFlagQR     = 1 << 15
	dnsFlagAA     = 1 << 10
	dnsFlagRD     = 1 << 8
	dnsOpcodeMask = 0x7800

	dnsRCodeSuccess       = 0
	dnsRCodeFormatError   = 1
	dnsRCodeServerFailure = 2
	dnsRCodeNameError     = 3
)

type Server struct {
	store  *state.Store
	cfg    config.Config
	logger *telemetry.Logger

	mu   sync.Mutex
	conn net.PacketConn
}

type query struct {
	id           uint16
	flags        uint16
	name         string
	qtype        uint16
	qclass       uint16
	questionWire []byte
}

func NewServer(store *state.Store, cfg config.Config, logger *telemetry.Logger) *Server {
	return &Server{store: store, cfg: cfg, logger: logger}
}

func (s *Server) Addr() string {
	return s.cfg.ListenHost + ":" + s.cfg.DNS
}

func (s *Server) ListenAndServe() error {
	conn, err := net.ListenPacket("udp", s.Addr())
	if err != nil {
		return fmt.Errorf("listen dns: %w", err)
	}

	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()
	defer s.Close()

	buf := make([]byte, 1500)
	for {
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("read dns packet: %w", err)
		}

		request := append([]byte(nil), buf[:n]...)
		response := s.handleQuery(request)
		if len(response) == 0 {
			continue
		}
		if _, err := conn.WriteTo(response, addr); err != nil && !errors.Is(err, net.ErrClosed) {
			s.logger.Info("tinycloud dns write failed", map[string]any{"error": err.Error()})
		}
	}
}

func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn == nil {
		return nil
	}
	err := s.conn.Close()
	s.conn = nil
	return err
}

func (s *Server) handleQuery(packet []byte) []byte {
	q, err := parseQuery(packet)
	if err != nil {
		return formatErrorResponse(packet)
	}
	if q.qclass != dnsClassIN {
		return buildResponse(q, nil, dnsRCodeSuccess)
	}
	if q.qtype != dnsTypeA {
		return buildResponse(q, nil, dnsRCodeSuccess)
	}

	recordSet, err := s.store.ResolvePrivateDNSARecordSet(q.name)
	if errors.Is(err, sql.ErrNoRows) {
		return buildResponse(q, nil, dnsRCodeNameError)
	}
	if err != nil {
		return buildResponse(q, nil, dnsRCodeServerFailure)
	}
	return buildResponse(q, recordSet.IPv4Addresses, dnsRCodeSuccess, uint32(recordSet.TTL))
}

func parseQuery(packet []byte) (query, error) {
	if len(packet) < 12 {
		return query{}, errors.New("dns packet too short")
	}

	qdCount := binary.BigEndian.Uint16(packet[4:6])
	if qdCount != 1 {
		return query{}, errors.New("dns packet must contain exactly one question")
	}

	offset := 12
	labels := make([]string, 0, 4)
	for {
		if offset >= len(packet) {
			return query{}, errors.New("dns packet ended inside qname")
		}
		length := int(packet[offset])
		offset++
		if length == 0 {
			break
		}
		if offset+length > len(packet) {
			return query{}, errors.New("dns packet label exceeds packet size")
		}
		labels = append(labels, string(packet[offset:offset+length]))
		offset += length
	}

	if offset+4 > len(packet) {
		return query{}, errors.New("dns packet missing qtype or qclass")
	}

	return query{
		id:           binary.BigEndian.Uint16(packet[0:2]),
		flags:        binary.BigEndian.Uint16(packet[2:4]),
		name:         stateName(labels),
		qtype:        binary.BigEndian.Uint16(packet[offset : offset+2]),
		qclass:       binary.BigEndian.Uint16(packet[offset+2 : offset+4]),
		questionWire: append([]byte(nil), packet[12:offset+4]...),
	}, nil
}

func formatErrorResponse(packet []byte) []byte {
	response := make([]byte, 12)
	if len(packet) >= 2 {
		copy(response[:2], packet[:2])
	}
	binary.BigEndian.PutUint16(response[2:4], dnsFlagQR|dnsFlagAA|dnsRCodeFormatError)
	return response
}

func buildResponse(q query, ipv4Addresses []string, rcode uint16, ttl ...uint32) []byte {
	answers := make([][4]byte, 0, len(ipv4Addresses))
	for _, address := range ipv4Addresses {
		ip := net.ParseIP(address).To4()
		if ip == nil {
			continue
		}
		var raw [4]byte
		copy(raw[:], ip)
		answers = append(answers, raw)
	}

	responseTTL := uint32(300)
	if len(ttl) > 0 && ttl[0] > 0 {
		responseTTL = ttl[0]
	}

	flags := dnsFlagQR | dnsFlagAA | (q.flags & (dnsFlagRD | dnsOpcodeMask)) | rcode
	response := make([]byte, 12, 12+len(q.questionWire)+(len(answers)*16))
	binary.BigEndian.PutUint16(response[0:2], q.id)
	binary.BigEndian.PutUint16(response[2:4], flags)
	binary.BigEndian.PutUint16(response[4:6], 1)
	binary.BigEndian.PutUint16(response[6:8], uint16(len(answers)))
	binary.BigEndian.PutUint16(response[8:10], 0)
	binary.BigEndian.PutUint16(response[10:12], 0)
	response = append(response, q.questionWire...)

	for _, answer := range answers {
		response = append(response, 0xc0, 0x0c)
		response = binary.BigEndian.AppendUint16(response, dnsTypeA)
		response = binary.BigEndian.AppendUint16(response, dnsClassIN)
		response = binary.BigEndian.AppendUint32(response, responseTTL)
		response = binary.BigEndian.AppendUint16(response, 4)
		response = append(response, answer[:]...)
	}

	return response
}

func stateName(labels []string) string {
	name := ""
	for i, label := range labels {
		if i > 0 {
			name += "."
		}
		name += label
	}
	return name
}
