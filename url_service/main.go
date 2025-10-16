package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/my_projects/url_service/configs"
	"github.com/my_projects/url_service/gen"
	"github.com/my_projects/url_service/pgx"
	"github.com/my_projects/url_service/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	ctx := context.Background()

	// Load configuration
	cfg, err := configs.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create database connection pool
	dbURL := cfg.Database.GetDatabaseURL()
	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("Failed to parse database URL: %v", err)
	}

	poolConfig.MaxConns = int32(cfg.Database.MaxConnections)
	poolConfig.MinConns = int32(cfg.Database.MaxIdleConnections)

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatalf("Failed to create database pool: %v", err)
	}
	defer pool.Close()

	// Test database connection
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Successfully connected to database")

	// Initialize repository
	urlRepo := pgx.NewUrlRepository(pool)

	// Initialize service with repository
	urlService := service.NewURLServiceImpl(urlRepo)

	// Create gRPC server
	serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	listener, err := net.Listen("tcp", serverAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", serverAddr, err)
	}

	grpcServer := grpc.NewServer()
	gen.RegisterUrlServiceServer(grpcServer, urlService)

	// Start gRPC server in a goroutine
	go func() {
		log.Printf("Starting gRPC server on %s", serverAddr)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Create HTTP gateway
	httpPort := cfg.Server.Port + 1 // HTTP on port 8081 by default
	httpAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, httpPort)

	// Create gRPC-Gateway mux
	gwmux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	// Register the gateway with the gRPC endpoint
	err = gen.RegisterUrlServiceHandlerFromEndpoint(ctx, gwmux, serverAddr, opts)
	if err != nil {
		log.Fatalf("Failed to register gateway: %v", err)
	}

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    httpAddr,
		Handler: gwmux,
	}

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("Starting HTTP gateway server on %s", httpAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to serve HTTP: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down gracefully...")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	} else {
		log.Println("HTTP server stopped gracefully")
	}

	// Graceful shutdown of gRPC server
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	// Wait for graceful shutdown or force stop after remaining timeout
	select {
	case <-stopped:
		log.Println("gRPC server stopped gracefully")
	case <-shutdownCtx.Done():
		grpcServer.Stop()
		log.Println("gRPC server force stopped after timeout")
	}

	// Close database pool
	pool.Close()
	log.Println("Database connection closed")
}
