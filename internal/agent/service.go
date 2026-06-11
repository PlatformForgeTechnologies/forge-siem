package agent

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"forge-siem/internal/config"
	"forge-siem/internal/types"
)

type Service struct {
	cfg       config.AppConfig
	agentID   string
	fileCfg   config.AgentFileConfig
	offsets   map[string]int64
	offsetsMu sync.Mutex
}

type logEntry struct {
	Line       string
	NextOffset int64
}

func New(cfg config.AppConfig) *Service {
	configPath := getenvOr("AGENT_CONFIG_PATH", "agent.yaml")
	fileCfg, err := config.LoadAgentFile(configPath)
	if err != nil {
		log.Printf("agent config load failed, falling back to defaults: %v", err)
	}

	return &Service{
		cfg:     cfg,
		agentID: getenvOr("AGENT_ID", uuid.NewString()),
		fileCfg: fileCfg,
		offsets: map[string]int64{},
	}
}

func (s *Service) Run(ctx context.Context) error {
	heartbeatTicker := time.NewTicker(30 * time.Second)
	logTicker := time.NewTicker(500 * time.Millisecond)
	defer heartbeatTicker.Stop()
	defer logTicker.Stop()

	log.Printf("agent %s started, target=%s:%d", s.agentID, s.fileCfg.Server.Host, s.fileCfg.Server.Port)
	var conn net.Conn

	for {
		select {
		case <-ctx.Done():
			if conn != nil {
				_ = conn.Close()
			}
			return nil
		case <-heartbeatTicker.C:
			var err error
			conn, err = s.ensureConn(conn)
			if err != nil {
				log.Printf("connect failed before heartbeat: %v", err)
				continue
			}
			if err := s.sendHeartbeat(conn); err != nil {
				log.Printf("heartbeat failed: %v", err)
				conn = s.resetConn(conn)
			}
		case <-logTicker.C:
			for _, pattern := range s.fileCfg.LogCollection.Paths {
				matches, _ := filepath.Glob(pattern)
				for _, path := range matches {
					entries, err := s.readNewEntries(path)
					if err != nil {
						log.Printf("log read failed for %s: %v", path, err)
						continue
					}
					for _, entry := range entries {
						conn, err = s.ensureConn(conn)
						if err != nil {
							log.Printf("connect failed before shipping %s: %v", path, err)
							break
						}
						if err := s.sendLog(conn, path, entry.Line); err != nil {
							log.Printf("ship log failed: %v", err)
							conn = s.resetConn(conn)
							break
						}
						s.setFileOffset(path, entry.NextOffset)
					}
				}
			}
		}
	}
}

func hostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return name
}

func getenvOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func ValidateFIMPath(path string) error {
	for _, excluded := range []string{"/proc", "/sys", "/dev", "/run"} {
		if path == excluded || strings.HasPrefix(path, excluded+"/") {
			return fmt.Errorf("path %s is hard-excluded from FIM", path)
		}
	}
	return nil
}

func (s *Service) connect() (net.Conn, error) {
	cert, err := tls.LoadX509KeyPair(s.fileCfg.Server.ClientCert, s.fileCfg.Server.ClientKey)
	if err != nil {
		return nil, fmt.Errorf("load client certificate: %w", err)
	}
	caPEM, err := os.ReadFile(s.fileCfg.Server.CACert)
	if err != nil {
		return nil, fmt.Errorf("read CA certificate: %w", err)
	}
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caPEM)

	address := fmt.Sprintf("%s:%d", s.fileCfg.Server.Host, s.fileCfg.Server.Port)
	conn, err := tls.Dial("tcp", address, &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		MinVersion:   tls.VersionTLS13,
	})
	if err != nil {
		return nil, fmt.Errorf("dial ingest %s: %w", address, err)
	}
	return conn, nil
}

func (s *Service) ensureConn(conn net.Conn) (net.Conn, error) {
	if conn != nil {
		return conn, nil
	}
	conn, err := s.connect()
	if err != nil {
		return nil, err
	}
	if err := s.sendHeartbeat(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

func (s *Service) resetConn(conn net.Conn) net.Conn {
	if conn != nil {
		_ = conn.Close()
	}
	return nil
}

func (s *Service) sendHeartbeat(conn net.Conn) error {
	hb := types.AgentHeartbeat{
		AgentID:   s.agentID,
		Hostname:  hostname(),
		IP:        os.Getenv("AGENT_IP"),
		OS:        runtime.GOOS,
		Kernel:    os.Getenv("AGENT_KERNEL"),
		Uptime:    time.Now().Unix(),
		Timestamp: time.Now().UTC(),
	}
	return s.writeEnvelope(conn, types.Envelope{
		Type:      types.EnvelopeTypeHeartbeat,
		AgentID:   s.agentID,
		Timestamp: time.Now().UTC(),
		Payload: map[string]any{
			"heartbeat": hb,
		},
	})
}

func (s *Service) sendLog(conn net.Conn, path, line string) error {
	if strings.TrimSpace(line) == "" {
		return nil
	}
	return s.writeEnvelope(conn, types.Envelope{
		Type:      types.EnvelopeTypeLog,
		AgentID:   s.agentID,
		Timestamp: time.Now().UTC(),
		Payload: map[string]any{
			"path": path,
			"line": line,
		},
	})
}

func (s *Service) writeEnvelope(conn net.Conn, env types.Envelope) error {
	body, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}
	if _, err := conn.Write(append(body, '\n')); err != nil {
		return fmt.Errorf("write envelope: %w", err)
	}
	return nil
}

func (s *Service) readNewEntries(path string) ([]logEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	offset := s.fileOffset(path)
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}

	reader := bufio.NewReader(file)
	var entries []logEntry
	var bytesRead int64
	for {
		line, err := reader.ReadString('\n')
		bytesRead += int64(len(line))
		if line != "" {
			entries = append(entries, logEntry{
				Line:       strings.TrimRight(line, "\r\n"),
				NextOffset: offset + bytesRead,
			})
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}
	return entries, nil
}

func (s *Service) fileOffset(path string) int64 {
	s.offsetsMu.Lock()
	defer s.offsetsMu.Unlock()
	return s.offsets[path]
}

func (s *Service) setFileOffset(path string, offset int64) {
	s.offsetsMu.Lock()
	defer s.offsetsMu.Unlock()
	s.offsets[path] = offset
}
