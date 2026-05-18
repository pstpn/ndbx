package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof" //nolint:gosec // for debug app only
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"ndbx/config"
	"ndbx/internal/router"
	oas "ndbx/internal/router/ogen"
	"ndbx/internal/service"
	cstorage "ndbx/internal/storage/cassandra"
	mstorage "ndbx/internal/storage/mongodb"
	nstorage "ndbx/internal/storage/neo4j"
	rstorage "ndbx/internal/storage/redis"
	"ndbx/pkg/cassandra"
	"ndbx/pkg/httpserver"
	"ndbx/pkg/logger"
	"ndbx/pkg/mongodb"
	neo4jpkg "ndbx/pkg/neo4j"
	"ndbx/pkg/redis"
)

func Run(ctx context.Context, cfg *config.Config) error {
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	l := logger.New(cfg.LogLevel)
	gr, ctx := errgroup.WithContext(ctx)

	redisClient, err := redis.NewClient(ctx, redisAddr(cfg.RedisHost, cfg.RedisPort), cfg.RedisDB, cfg.RedisPassword)
	if err != nil {
		return fmt.Errorf("new redis client: %w", err)
	}
	defer redisClient.Close()

	mongoDBClient, err := mongodb.New(
		ctx,
		cfg.MongoDBUser,
		cfg.MongoDBPassword,
		cfg.MongoDBHost,
		cfg.MongoDBPort,
		cfg.MongoDBDatabase,
	)
	if err != nil {
		return fmt.Errorf("new mongodb client: %w", err)
	}
	defer mongoDBClient.Close(ctx)

	cassandraClient, err := cassandra.NewClient(
		ctx,
		strings.Split(cfg.CassandraHosts, ","),
		cfg.CassandraPort,
		cfg.CassandraUsername,
		cfg.CassandraPassword,
		cfg.CassandraKeyspace,
		cfg.CassandraConsistency,
	)
	if err != nil {
		return fmt.Errorf("new cassandra client: %w", err)
	}
	defer cassandraClient.Close()

	neo4jClient, err := neo4jpkg.NewClient(ctx, cfg.Neo4jURL, cfg.Neo4jUser, cfg.Neo4jPassword)
	if err != nil {
		return fmt.Errorf("new neo4j client: %w", err)
	}
	defer neo4jClient.Close(ctx)

	// Storages
	sessionStorage := rstorage.NewSessionStorage(redisClient)
	reactionCacheStorage := rstorage.NewEventReactionStorage(redisClient)
	reviewCacheStorage := rstorage.NewEventReviewCacheStorage(redisClient)
	recommendationCacheStorage := rstorage.NewRecommendationCacheStorage(redisClient)
	userStorage := mstorage.NewUserStorage(mongoDBClient.DB())
	eventStorage := mstorage.NewEventStorage(mongoDBClient.DB())
	reactionStorage := cstorage.NewEventReactionStorage(cassandraClient.Session())
	reviewStorage := cstorage.NewEventReviewStorage(cassandraClient.Session())
	graphStorage := nstorage.NewGraphStorage(neo4jClient.Driver())

	if err := userStorage.CreateIndex(ctx); err != nil {
		return fmt.Errorf("create user indexes: %w", err)
	}
	if err := eventStorage.CreateIndexes(ctx); err != nil {
		return fmt.Errorf("create event indexes: %w", err)
	}

	// Services
	sessionService := service.NewSessionService(l, sessionStorage, cfg.AppUserSessionTTLSeconds)
	userService := service.NewUserService(l, userStorage, graphStorage)
	reviewService := service.NewReviewService(l, eventStorage, reviewStorage, reviewCacheStorage, cfg.AppEventReviewsTTLSeconds)
	eventService := service.NewEventService(l, eventStorage, reactionStorage, reactionCacheStorage, reviewService, graphStorage, cfg.AppLikeTTLSeconds)
	recommendationService := service.NewRecommendationService(l, graphStorage, eventStorage, recommendationCacheStorage, cfg.AppRecommendationsTTLSeconds)

	handler := router.NewHandler(l, sessionService, userService, eventService, reviewService, recommendationService, cfg.AppUserSessionTTLSeconds)

	oasHandler, err := oas.NewServer(handler)
	if err != nil {
		return fmt.Errorf("new oas handler: %w", err)
	}

	httpAddr := fmt.Sprintf("%s:%d", cfg.HTTPHost, cfg.HTTPPort)
	httpServer := httpserver.NewServer(l)
	pprofAddr := fmt.Sprintf("%s:%d", cfg.HTTPHost, cfg.PprofPort)
	pprofServer := http.Server{Addr: pprofAddr, ReadHeaderTimeout: time.Second}

	gr.Go(func() error {
		l.Infof("starting http server on %s", httpAddr)
		return httpServer.Run(httpAddr,
			httpserver.Wrap(
				oasHandler,
				httpserver.CORSMiddleware(l),
				httpserver.DocsMiddleware(l),
			),
		)
	})
	gr.Go(func() error {
		l.Infof("starting pprof server on %s", pprofAddr)
		if err := pprofServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			l.Errorf("pprof server stopped with error: %v", err)
			return err
		}

		l.Infof("pprof server stopped")
		return nil
	})

	gr.Go(func() error {
		<-ctx.Done()
		httpServer.Shutdown()
		return pprofServer.Shutdown(ctx)
	})

	return gr.Wait()
}

func redisAddr(host string, port int) string {
	return fmt.Sprintf("%s:%d", host, port)
}
