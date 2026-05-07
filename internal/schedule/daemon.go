package schedule

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type runnerHandle struct {
	cancel context.CancelFunc
	done   chan struct{}
}

// RunDaemon is the entry point for the detached daemon process.
func RunDaemon() {
	if err := writePID(os.Getpid()); err != nil {
		log.Printf("daemon: failed to write PID: %v", err)
	}
	defer removePID()

	log.Printf("daemon: started (pid %d)", os.Getpid())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runners := map[string]*runnerHandle{}
	var mu sync.Mutex

	statusCh := make(chan StatusUpdate, 64)

	// Status writer goroutine — serializes all registry status updates.
	go func() {
		for upd := range statusCh {
			ApplyStatus(upd.ID, upd.LastRun, upd.NextRun, upd.Err)
		}
	}()

	reload := func() {
		reg, err := LoadRegistry()
		if err != nil {
			log.Printf("daemon: reload error: %v", err)
			return
		}
		schedules := reg.Snapshot()
		mu.Lock()
		defer mu.Unlock()

		// Index current schedules.
		wanted := map[string]Schedule{}
		for _, s := range schedules {
			wanted[s.ID] = s
		}

		// Stop runners no longer in registry.
		for id, h := range runners {
			if _, ok := wanted[id]; !ok {
				log.Printf("daemon: stopping runner %s", id)
				h.cancel()
				<-h.done
				delete(runners, id)
			}
		}

		// Start new runners or restart if interval changed.
		for id, s := range wanted {
			if h, exists := runners[id]; exists {
				// Check if interval changed — naive: compare by ID presence only for now.
				_ = h
				continue
			}
			log.Printf("daemon: starting runner %s (%s/%s every %s)", id, s.Owner, s.Name, s.Interval)
			rctx, rcancel := context.WithCancel(ctx)
			done := make(chan struct{})
			sCopy := s
			go func() {
				defer close(done)
				Run(rctx, sCopy, statusCh)
			}()
			runners[id] = &runnerHandle{cancel: rcancel, done: done}
		}

		if len(schedules) == 0 {
			log.Printf("daemon: no schedules registered, exiting")
			cancel()
		}
	}

	reload()

	sigs := make(chan os.Signal, 4)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	for {
		select {
		case <-ctx.Done():
			// Graceful shutdown: wait for all runners.
			mu.Lock()
			for id, h := range runners {
				log.Printf("daemon: waiting for runner %s", id)
				h.cancel()
				select {
				case <-h.done:
				case <-time.After(5 * time.Second):
					log.Printf("daemon: runner %s did not stop in time", id)
				}
			}
			mu.Unlock()
			close(statusCh)
			log.Printf("daemon: stopped")
			return

		case sig := <-sigs:
			switch sig {
			case syscall.SIGHUP:
				log.Printf("daemon: SIGHUP — reloading registry")
				reload()
			case syscall.SIGTERM, syscall.SIGINT:
				log.Printf("daemon: received %s — shutting down", sig)
				cancel()
			}
		}
	}
}
