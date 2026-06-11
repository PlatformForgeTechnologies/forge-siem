package ingest

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/redis/go-redis/v9"

	"forge-siem/internal/config"
	"forge-siem/internal/platform"
	"forge-siem/internal/types"
)

type Service struct {
	cfg config.AppConfig
}

func New(cfg config.AppConfig) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) Run(ctx context.Context) error {
	cert, err := tls.LoadX509KeyPair(s.cfg.TLSCertFile, s.cfg.TLSKeyFile)
	if err != nil {
		return fmt.Errorf("load ingest certificate: %w", err)
	}
	caPEM, err := os.ReadFile(s.cfg.TLSCAFile)
	if err != nil {
		return fmt.Errorf("read ingest CA: %w", err)
	}
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caPEM)

	listener, err := tls.Listen("tcp", ":1514", &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
		MinVersion:   tls.VersionTLS13,
	})
	if err != nil {
		return fmt.Errorf("listen on :1514: %w", err)
	}
	defer listener.Close()

	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	log.Printf("ingest listening on :1514 using Redis stream %s", s.cfg.StreamRaw)
	redisClient := platform.NewRedis(s.cfg)
	defer redisClient.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("accept failed: %v", err)
			continue
		}
		go s.handleConn(ctx, redisClient, conn)
	}
}

func (s *Service) handleConn(ctx context.Context, redisClient *platform.Redis, conn net.Conn) {
	defer conn.Close()
	remote := conn.RemoteAddr().String()
	log.Printf("accepted agent connection from %s, forwarding events to %s", remote, s.cfg.StreamRaw)

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Minute))
		var env types.Envelope
		if err := json.Unmarshal(scanner.Bytes(), &env); err != nil {
			log.Printf("invalid envelope from %s: %v", remote, err)
			continue
		}
		if err := s.acceptEnvelope(ctx, redisClient, env, remote); err != nil {
			log.Printf("ingest failed for %s: %v", remote, err)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("connection scan failed for %s: %v", remote, err)
	}
}

func (s *Service) acceptEnvelope(ctx context.Context, redisClient *platform.Redis, env types.Envelope, remote string) error {
	payload := map[string]any{
		"remote_addr": remote,
		"received_at": time.Now().UTC(),
		"type":        env.Type,
		"agent_id":    env.AgentID,
		"timestamp":   env.Timestamp,
		"payload":     env.Payload,
	}
	values, err := platform.MarshalMap(payload)
	if err != nil {
		return err
	}
	if err := redisClient.Client().XAdd(ctx, &redis.XAddArgs{
		Stream: s.cfg.StreamRaw,
		Values: values,
	}).Err(); err != nil {
		return fmt.Errorf("xadd raw event: %w", err)
	}
	if env.Type == types.EnvelopeTypeHeartbeat {
		key := fmt.Sprintf("agent:%s:heartbeat", env.AgentID)
		if err := redisClient.Client().Set(ctx, key, time.Now().UTC().Format(time.RFC3339), s.cfg.HeartbeatTTL).Err(); err != nil {
			return fmt.Errorf("set heartbeat ttl: %w", err)
		}
	}
	return nil
}
