package main

import (
	"context"
	"glooko/internal/adapters/mongodb"
	"glooko/internal/api"
	"glooko/internal/config"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := zap.NewProduction()
	log := logger.Sugar()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("failed to load config", zap.Error(err))
	}

	log.Infof("config: %+v", cfg)

	mongoDB, err := mongodb.NewMongoDB(ctx, cfg.MongoDBURI, cfg.MongoDBName)
	if err != nil {
		log.Fatal("failed to connect to MongoDB", zap.Error(err))
	}

	userRepository := mongodb.NewUserRepository(mongoDB)
	deviceRepository := mongodb.NewDeviceRepository(mongoDB)
	readingsRepository := mongodb.NewReadingRepository(mongoDB)

	mainAPI := api.NewAPI(log, userRepository, deviceRepository, readingsRepository)

	server := &http.Server{
		Addr:    cfg.ServerPort,
		Handler: mainAPI.Routes(),
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sig

		shutdownCtx, _ := context.WithTimeout(ctx, 30*time.Second)

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				log.Fatal("graceful shutdown timed out.. forcing exit")
			}
		}()

		err := server.Shutdown(shutdownCtx)
		if err != nil {
			log.Fatal(err)
		}
		cancel()
	}()

	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

	log.Info("server running on port", cfg.ServerPort)

	<-ctx.Done()
}
