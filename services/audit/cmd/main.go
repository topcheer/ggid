// Package main is the entry point for the Audit Service.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/ggid/ggid/api/gen/audit/v1"

	"github.com/ggid/ggid/pkg/audit"
	"github.com/ggid/ggid/pkg/middleware"
	"github.com/ggid/ggid/services/audit/internal/alerting"
	"github.com/ggid/ggid/services/audit/internal/config"
	"github.com/ggid/ggid/services/audit/internal/consumer"
	"github.com/ggid/ggid/services/audit/internal/data"
	"github.com/ggid/ggid/services/audit/internal/detection"
	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/ggid/ggid/services/audit/internal/handler"
	"github.com/ggid/ggid/services/audit/internal/repository"
	httpserver "github.com/ggid/ggid/services/audit/internal/server"
	"github.com/ggid/ggid/services/audit/internal/service"
	"github.com/ggid/ggid/pkg/shutdown"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// newGRPCServer creates a gRPC server with optional TLS based on GRPC_TLS_ENABLED env var.
func newGRPCServer() *grpc.Server {
	if os.Getenv("GRPC_TLS_ENABLED") == "true" {
		certFile := os.Getenv("GRPC_TLS_CERT")
		keyFile := os.Getenv("GRPC_TLS_KEY")
		if certFile != "" && keyFile != "" {
			tlsCfg, err := tls.LoadX509KeyPair(certFile, keyFile)
			if err == nil {
				return grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{
					Certificates: []tls.Certificate{tlsCfg},
					MinVersion:   tls.VersionTLS12,
				})))
			}
			if os.Getenv("GRPC_TLS_ALLOW_PLAINTEXT_FALLBACK") != "true" {
				log.Fatalf("GRPC_TLS_ENABLED but cert/key invalid: %v; refusing to start with plaintext fallback. Set GRPC_TLS_ALLOW_PLAINTEXT_FALLBACK=true only in dev.", err)
			}
			log.Printf("Warning: GRPC_TLS_ENABLED but cert/key invalid: %v, falling back to plaintext because GRPC_TLS_ALLOW_PLAINTEXT_FALLBACK=true", err)
		}
	}
	return grpc.NewServer()
}

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "Run migrations only and exit")
	noConsumer := flag.Bool("no-consumer", false, "Disable NATS consumer (query-only mode)")
	flag.Parse()

	cfg := config.FromEnv()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := data.New(ctx, cfg.DB)
	if err != nil {
		cancel()
		log.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()
	log.Println("Audit Service: database connected")

	// Initialize hash chain for tamper-evident audit events
	if cfg.HashChainSecret != "" {
		domain.SetHashChainSecret([]byte(cfg.HashChainSecret))
		log.Println("Audit Service: hash chain enabled")
	} else {
		log.Println("Audit Service: WARNING — hash chain disabled (set AUDIT_HASH_CHAIN_SECRET to enable)")
	}

	// Initialize repository and service
	repo := repository.NewAuditRepository(db)
	auditSvc := service.NewAuditService(repo)

	if *migrateOnly {
		log.Println("Audit Service: migration mode, skipping server start")
		return
	}

	// Start NATS consumer (unless disabled)
	var natsConsumer *consumer.EventConsumer
	if !*noConsumer {
		nc, err := consumer.New(ctx, cfg.NATS, repo)
		if err != nil {
			log.Fatalf("failed to create NATS consumer: %v", err)
		}

		// Wire ITDR detection engine into the consumer loop.
		if db != nil {
			engineRepo := repository.NewITDRRepository(db)

			// Use Redis StateStore for multi-replica safety. Fall back to MemStateStore if Redis unavailable.
			var stateStore detection.StateStore
			if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
				opts, err := redis.ParseURL(redisURL)
				if err == nil {
					rdb := redis.NewClient(opts)
					stateStore = detection.NewRedisStateStore(rdb)
					log.Println("Audit Service: ITDR using Redis StateStore (multi-replica safe)")
				}
			}
			if stateStore == nil {
				stateStore = detection.NewMemStateStore()
				log.Println("Audit Service: ITDR using MemStateStore (single-replica only)")
			}

			engine := detection.NewEngine(engineRepo, stateStore)
			detection.RegisterKB192Rules(engine.Registry())
			nc.SetEngine(engine)
			log.Println("Audit Service: ITDR detection engine enabled")
		}

		if err := nc.Start(); err != nil {
			log.Fatalf("failed to start NATS consumer: %v", err)
		}
		natsConsumer = nc
		log.Println("Audit Service: NATS consumer started")
	}

	// Initialize gRPC handler
	auditHandler := handler.NewAuditHandler(auditSvc)

	// Start gRPC server (TLS-aware via GRPC_TLS_ENABLED env var)
	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", cfg.GRPCAddr, err)
	}
	grpcServer := newGRPCServer()
	pb.RegisterAuditServiceServer(grpcServer, auditHandler)

	go func() {
		log.Printf("Audit Service: gRPC listening on %s", cfg.GRPCAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC serve: %v", err)
		}
	}()

	// HTTP server (health + REST API for Console)
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	})
	// REST API endpoints
	httpAPI := httpserver.NewHTTPServer(auditSvc)
	if db != nil {
		httpAPI.SetPool(db)
	}

	// Wire ITDR repository into HTTP server for API queries.
	if db != nil {
		itdrRepo := repository.NewITDRRepository(db)
		// Ensure tables exist at startup.
		if err := itdrRepo.EnsureSchema(ctx); err != nil {
			log.Printf("Warning: ITDR EnsureSchema failed: %v", err)
		}
		httpAPI.SetITDRRepository(itdrRepo)

		// Memory map batch 2: integrity, webhook_deliveries, dsr_requests, collect_schedules, dedup
		mmRepo2 := httpserver.NewAuditMemoryMapRepo2(db)
		if err := mmRepo2.EnsureSchema(ctx); err != nil {
			log.Printf("Warning: MemoryMap2 EnsureSchema failed: %v", err)
		}
		httpAPI.SetMemMapRepo2(mmRepo2)

		// Composite detection rules (Task-C): PG-backed persistence replacing
		// the in-memory map so rules survive restarts and stay consistent
		// across replicas.
		compositeRepo := httpserver.NewCompositeRuleRepo(db)
		if err := compositeRepo.EnsureSchema(ctx); err != nil {
			log.Printf("Warning: CompositeRule EnsureSchema failed: %v", err)
		}
		httpAPI.SetCompositeRepo(compositeRepo)

		// Threat Intelligence Integration Hub (B-37).
		threatRepo := repository.NewThreatIntelRepository(db)
		if err := threatRepo.EnsureSchema(ctx); err != nil {
			log.Printf("Warning: ThreatIntel EnsureSchema failed: %v", err)
		}
		httpAPI.SetThreatIntelRepo(threatRepo)

		// CCM (Continuous Compliance Monitoring) repository (KB-280).
		ccmRepo := repository.NewCCMRepository(db)
		if err := ccmRepo.EnsureSchema(ctx); err != nil {
			log.Printf("Warning: CCM EnsureSchema failed: %v", err)
		}
		httpAPI.SetCCMRepository(ccmRepo)
		httpAPI.SetCCMPool(db) // KB-346: enable real DB queries in CCM engine

		// Start async collector goroutine.
		collector := httpserver.NewIntelCollector(threatRepo)
		go collector.Run(ctx, 10*time.Minute)

		log.Println("Audit Service: ITDR + Threat Intel API enabled")
}

	httpAPI.RegisterRoutes(mux)

	mwSecret, mwPrevSecret := middleware.LoadInternalSecrets()
	protectedMux := middleware.InternalAuthPathOnly(middleware.InternalAuthConfig{
		Secret:     mwSecret,
		PrevSecret: mwPrevSecret,
	})(mux)

	httpServer := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					log.Printf("PANIC recovered in audit handler: %v", rvr)
					http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				}
			}()
			protectedMux.ServeHTTP(w, r)
		}),
	}

	// SIEM Forwarder — forward audit events to external SIEM (Splunk/Datadog/Elasticsearch)
	var siemForwarder *audit.SIEMForwarder
	if siemEndpoint := os.Getenv("SIEM_ENDPOINT"); siemEndpoint != "" {
		siemProvider := audit.SIEMProvider(os.Getenv("SIEM_PROVIDER"))
		if siemProvider == "" {
			siemProvider = audit.SIEMProviderGeneric
		}
		siemCfg := audit.DefaultSIEMConfig()
		siemCfg.Provider = siemProvider
		siemCfg.Endpoint = siemEndpoint
		siemCfg.APIKey = os.Getenv("SIEM_API_KEY")
		siemCfg.IndexName = os.Getenv("SIEM_INDEX")
		siemForwarder = audit.NewSIEMForwarder(siemCfg)
		// Wire custom CA cert for SIEM TLS if provided
		if caCertPath := os.Getenv("SIEM_CA_CERT"); caCertPath != "" {
			if pemData, err := os.ReadFile(caCertPath); err == nil {
				caPool := x509.NewCertPool()
				if caPool.AppendCertsFromPEM(pemData) {
					siemForwarder.SetCAPool(caPool)
					log.Printf("SIEM: custom CA cert loaded from %s", caCertPath)
				}
			}
		}
		siemForwarder.Start(ctx)
		log.Printf("Audit Service: SIEM forwarder started (provider=%s, endpoint=%s)", siemProvider, siemEndpoint)
	}

	// Alert Engine — real-time alerting on audit events via NATS subscription.
	// Reads alert rules from ALERT_RULES_CONFIG (JSON file path).
	var alertEngine *alerting.AlertEngine
	var alertNc *nats.Conn
	if alertConfigPath := os.Getenv("ALERT_RULES_CONFIG"); alertConfigPath != "" {
		rulesData, err := os.ReadFile(alertConfigPath)
		if err != nil {
			log.Printf("Audit Service: failed to read alert rules config: %v", err)
		} else {
			var rules []*alerting.AlertRule
			if err := json.Unmarshal(rulesData, &rules); err != nil {
				log.Printf("Audit Service: failed to parse alert rules config: %v", err)
			} else {
				var notifier alerting.Notifier
				if webhookURL := os.Getenv("ALERT_WEBHOOK_URL"); webhookURL != "" {
					notifier = &alerting.WebhookNotifier{URL: webhookURL}
				}
				alertEngine = alerting.NewAlertEngine(notifier)
				for _, rule := range rules {
					alertEngine.AddRule(rule)
				}

				// Subscribe to NATS audit events for real-time evaluation.
				alertNc, err = nats.Connect(cfg.NATS.URL,
					nats.MaxReconnects(-1),
					nats.ReconnectWait(2*time.Second),
				)
				if err != nil {
					log.Printf("Audit Service: failed to connect NATS for alerting: %v", err)
				} else {
					subject := cfg.NATS.Subject
					_, err = alertNc.Subscribe(subject, func(m *nats.Msg) {
						var event domain.AuditEvent
						if err := json.Unmarshal(m.Data, &event); err != nil {
							return
						}
						userID := ""
						if event.ActorID != nil {
							userID = event.ActorID.String()
						}
						alertEngine.Evaluate(ctx, &alerting.AlertEvent{
							TenantID:  event.TenantID.String(),
							Action:    event.Action,
							UserID:    userID,
							IPAddress: event.IPAddress,
							Timestamp: event.CreatedAt,
							Fields:    event.Metadata,
						})
					})
					if err != nil {
						log.Printf("Audit Service: failed to subscribe for alerting: %v", err)
					} else {
						log.Printf("Audit Service: alert engine started (%d rules loaded, subject=%s)", len(rules), subject)
					}
				}
			}
		}
	}

	go func() {
		log.Printf("Audit Service: HTTP listening on %s", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Set shutdown flag so health checks return 503.
	shutdown.New(&shutdown.Resources{HTTPServer: httpServer}).Execute()

	log.Println("Audit Service: shutting down...")
	grpcServer.GracefulStop()
	if natsConsumer != nil {
		natsConsumer.Close()
	}
	if siemForwarder != nil {
		siemForwarder.Stop()
	}
	if alertNc != nil {
		alertNc.Close()
	}
	log.Println("Audit Service: stopped")
}