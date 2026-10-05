package live

import (
	"bytes"
	"context"
	"io"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestRunnerEmitsAndStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var output bytes.Buffer
	renders := 0
	animations := 0
	metrics := 0
	runner := Runner{
		AnimationInterval: 10 * time.Millisecond,
		MetricsInterval:   time.Hour,
		Output:            &output,
		UpdateMetrics: func() error {
			metrics++
			return nil
		},
		AdvanceAnimation: func() {
			animations++
		},
		RenderInitial: func() (string, error) {
			renders++
			return "initial", nil
		},
		RenderAnimation: func() (string, error) {
			renders++
			if renders == 2 {
				cancel()
			}
			return "animation", nil
		},
		RenderMetrics: func() (string, error) {
			return "metrics", nil
		},
	}

	if err := runner.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if renders != 2 || animations != 1 || metrics != 1 || output.String() != "initialanimation" {
		t.Fatalf("Run() renders = %d, animations = %d, metrics = %d, output = %q", renders, animations, metrics, output.String())
	}
}

func TestRunnerRedrawsOnResize(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	resizeEvents := make(chan os.Signal, 1)
	resizeEvents <- syscall.SIGWINCH
	var output bytes.Buffer
	runner := Runner{
		AnimationInterval: time.Hour,
		MetricsInterval:   time.Hour,
		Output:            &output,
		RenderInitial: func() (string, error) {
			return "initial", nil
		},
		ResizeEvents: resizeEvents,
		RenderResize: func() (string, error) {
			cancel()
			return "resized", nil
		},
	}

	if err := runner.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got, want := output.String(), "initialresized"; got != want {
		t.Fatalf("Run() output = %q, want %q", got, want)
	}
}

func TestRunnerIgnoresClosedResizeChannel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	resizeEvents := make(chan os.Signal)
	close(resizeEvents)
	animations, resizes := 0, 0
	runner := Runner{
		AnimationInterval: time.Millisecond,
		MetricsInterval:   time.Hour,
		Output:            io.Discard,
		ResizeEvents:      resizeEvents,
		RenderInitial:     func() (string, error) { return "initial", nil },
		RenderAnimation: func() (string, error) {
			animations++
			cancel()
			return "animation", nil
		},
		RenderResize: func() (string, error) {
			resizes++
			cancel()
			return "resize", nil
		},
	}
	if err := runner.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if resizes != 0 || animations == 0 {
		t.Fatalf("closed resize channel: %d resizes, %d animations", resizes, animations)
	}
}

func TestRunnerKeepsAnimatingDuringMetricUpdate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	metricsStarted := make(chan struct{})
	releaseMetrics := make(chan struct{})
	animationRendered := make(chan struct{}, 1)
	done := make(chan error, 1)
	updates := 0
	runner := Runner{
		AnimationInterval: 5 * time.Millisecond,
		MetricsInterval:   time.Millisecond,
		Output:            io.Discard,
		UpdateMetrics: func() error {
			updates++
			if updates == 1 {
				return nil
			}
			close(metricsStarted)
			<-releaseMetrics
			return nil
		},
		RenderInitial: func() (string, error) { return "initial", nil },
		RenderAnimation: func() (string, error) {
			select {
			case animationRendered <- struct{}{}:
			default:
			}
			return "animation", nil
		},
		RenderMetrics: func() (string, error) {
			cancel()
			return "metrics", nil
		},
	}

	go func() { done <- runner.Run(ctx) }()
	select {
	case <-metricsStarted:
	case <-time.After(time.Second):
		t.Fatal("metric update did not start")
	}
	select {
	case <-animationRendered:
	case <-time.After(time.Second):
		t.Fatal("animation did not render while metric update was blocked")
	}
	close(releaseMetrics)
	if err := <-done; err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}
