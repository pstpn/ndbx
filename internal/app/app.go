package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof" //nolint:gosec // for debug app only
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"ndbx/config"
	"ndbx/internal/router"
	oas "ndbx/internal/router/ogen"
	"ndbx/pkg/httpserver"
	"ndbx/pkg/logger"
)

func Run(ctx context.Context, cfg *config.Config) error {
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	l := logger.New(cfg.Level)
	gr, ctx := errgroup.WithContext(ctx)

	oasHandler, err := oas.NewServer(router.NewHandler(l))
	if err != nil {
		return fmt.Errorf("new oas handler: %w", err)
	}

	httpAddr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	httpServer := httpserver.NewServer(l)
	pprofAddr := fmt.Sprintf("%s:%d", cfg.Host, cfg.PprofPort)
	pprofServer := http.Server{Addr: pprofAddr, ReadHeaderTimeout: time.Second}

	gr.Go(func() error {
		l.Infof("starting http server on %s", httpAddr)
		return httpServer.Run(httpAddr,
			httpserver.Wrap(
				oasHandler,
				httpserver.CORSMiddleware(l),
				httpserver.DocsMiddleware(l),
				httpserver.HeartbeatMiddleware("/health"),
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
