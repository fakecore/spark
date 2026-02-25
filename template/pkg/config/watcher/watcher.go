package watcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"spark/internal/conf"
	"spark/pkg/clog"
	"spark/pkg/utils/oss"
	"spark/pkg/utils/provider"

	"github.com/fsnotify/fsnotify"
	kratosconfig "github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
)

type ConfigWatcher struct {
	logger         clog.Logger
	filePath       string
	watcher        *fsnotify.Watcher
	bc             *conf.Bootstrap
	ossProvider    oss.OssProvider
	dbProvider     provider.DatabaseProvider
	kvProvider     provider.KVStoreProvider
	mu             sync.RWMutex
	lastConfigHash string
	closed         chan struct{}
}

func NewConfigWatcher(
	logger clog.Logger,
	filePath string,
	bc *conf.Bootstrap,
	ossProvider oss.OssProvider,
	dbProvider provider.DatabaseProvider,
	kvProvider provider.KVStoreProvider,
) *ConfigWatcher {
	w := &ConfigWatcher{
		logger:      logger,
		filePath:    filePath,
		bc:          bc,
		ossProvider: ossProvider,
		dbProvider:  dbProvider,
		kvProvider:  kvProvider,
		closed:      make(chan struct{}),
	}
	w.lastConfigHash = hashConfig(bc)
	return w
}

func (w *ConfigWatcher) Start(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	w.watcher = watcher

	if err := watcher.Add(w.filePath); err != nil {
		watcher.Close()
		return fmt.Errorf("watch config file: %w", err)
	}

	w.logger.Info(ctx, "config watcher: started", clog.String("path", w.filePath))

	go w.run(ctx)

	return nil
}

func (w *ConfigWatcher) run(ctx context.Context) {
	debounceTimer := time.NewTimer(0)
	if !debounceTimer.Stop() {
		<-debounceTimer.C
	}
	debounceDuration := 1 * time.Second

	for {
		select {
		case <-w.closed:
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				if !debounceTimer.Stop() {
					select {
					case <-debounceTimer.C:
					default:
					}
				}
				debounceTimer.Reset(debounceDuration)
			}
		case <-debounceTimer.C:
			w.reloadConfig(ctx)
		}
	}
}

func (w *ConfigWatcher) reloadConfig(ctx context.Context) {
	w.mu.Lock()
	defer w.mu.Unlock()

	c := kratosconfig.New(
		kratosconfig.WithSource(
			file.NewSource(w.filePath),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		w.logger.Error(ctx, "config watcher: reload failed - load error", clog.Err(err))
		return
	}

	var newBc conf.Bootstrap
	if err := c.Scan(&newBc); err != nil {
		w.logger.Error(ctx, "config watcher: reload failed - scan error", clog.Err(err))
		return
	}

	newHash := hashConfig(&newBc)
	if newHash == w.lastConfigHash {
		return
	}

	w.logger.Info(ctx, "config watcher: config changed, reloading...")

	var updateErrors []error

	if w.ossProvider != nil {
		if err := w.ossProvider.Update(ctx, newBc.Oss); err != nil {
			updateErrors = append(updateErrors, fmt.Errorf("OSS: %w", err))
			w.logger.Error(ctx, "config watcher: OSS update failed", clog.Err(err))
		}
	}

	if w.dbProvider != nil {
		if err := w.dbProvider.Update(ctx, newBc.Data.Database); err != nil {
			updateErrors = append(updateErrors, fmt.Errorf("Database: %w", err))
			w.logger.Error(ctx, "config watcher: Database update failed", clog.Err(err))
		}
	}

	if w.kvProvider != nil {
		if err := w.kvProvider.Update(ctx, newBc.Data.Redis); err != nil {
			updateErrors = append(updateErrors, fmt.Errorf("KVStore: %w", err))
			w.logger.Error(ctx, "config watcher: KVStore update failed", clog.Err(err))
		}
	}

	if len(updateErrors) > 0 {
		w.logger.Warn(ctx, "config watcher: some providers failed to update, keeping old config")
		return
	}

	w.bc = &newBc
	w.lastConfigHash = newHash
	w.logger.Info(ctx, "config watcher: reload completed successfully")
}

func (w *ConfigWatcher) Stop(ctx context.Context) error {
	close(w.closed)
	if w.watcher != nil {
		return w.watcher.Close()
	}
	return nil
}

func hashConfig(bc *conf.Bootstrap) string {
	return fmt.Sprintf("%v", bc)
}
