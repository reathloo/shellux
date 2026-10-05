package live

import (
	"context"
	"io"
	"os"
	"time"
)

// Runner periodically refreshes a rendered view until its context is done.
type Runner struct {
	AnimationInterval time.Duration
	MetricsInterval   time.Duration
	UpdateMetrics     func() error
	AdvanceAnimation  func()
	RenderInitial     func() (string, error)
	RenderAnimation   func() (string, error)
	RenderMetrics     func() (string, error)
	RenderResize      func() (string, error)
	ResizeEvents      <-chan os.Signal
	Output            io.Writer
}

// Run emits an initial view, then advances animation and metrics on independent
// schedules. The caller controls terminal escape sequences in Render's output.
func (r Runner) Run(ctx context.Context) error {
	if r.AnimationInterval <= 0 {
		r.AnimationInterval = 350 * time.Millisecond
	}
	if r.MetricsInterval <= 0 {
		r.MetricsInterval = time.Second
	}
	if r.RenderInitial == nil || r.Output == nil {
		return nil
	}

	write := func(render func() (string, error)) error {
		if render == nil {
			return nil
		}
		view, err := render()
		if err != nil {
			return err
		}
		_, err = io.WriteString(r.Output, view)
		return err
	}
	if r.UpdateMetrics != nil {
		if err := r.UpdateMetrics(); err != nil {
			return err
		}
	}
	if err := write(r.RenderInitial); err != nil {
		return err
	}

	animationTicker := time.NewTicker(r.AnimationInterval)
	defer animationTicker.Stop()
	metricsTicker := time.NewTicker(r.MetricsInterval)
	defer metricsTicker.Stop()
	metricsDone := make(chan error, 1)
	metricsRunning := false
	updateMetrics := func() {
		metricsRunning = true
		go func() {
			metricsDone <- r.UpdateMetrics()
		}()
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-animationTicker.C:
			if r.AdvanceAnimation != nil {
				r.AdvanceAnimation()
			}
			if err := write(r.RenderAnimation); err != nil {
				return err
			}
		case <-metricsTicker.C:
			if r.UpdateMetrics != nil && !metricsRunning {
				updateMetrics()
			}
		case err := <-metricsDone:
			metricsRunning = false
			if err != nil {
				return err
			}
			if err := write(r.RenderMetrics); err != nil {
				return err
			}
		case _, open := <-r.ResizeEvents:
			if !open {
				r.ResizeEvents = nil
				continue
			}
			if err := write(r.RenderResize); err != nil {
				return err
			}
		}
	}
}
