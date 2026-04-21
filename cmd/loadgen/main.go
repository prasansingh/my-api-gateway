package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

type headerFlags []string

func (h *headerFlags) String() string {
	return strings.Join(*h, ", ")
}

func (h *headerFlags) Set(value string) error {
	*h = append(*h, value)
	return nil
}

type Result struct {
	StatusCode int
	Latency    time.Duration
	Error      error
	Timestamp  time.Time
}

func main() {
	url := flag.String("url", "", "Target URL (required)")
	method := flag.String("method", "GET", "HTTP method")
	n := flag.Int("n", 100, "Total requests (ignored if -duration is set)")
	c := flag.Int("c", 10, "Concurrent workers")
	rate := flag.Int("rate", 0, "Requests per second limit (0 = unlimited)")
	duration := flag.Duration("duration", 0, "Run duration (overrides -n)")
	timeout := flag.Duration("timeout", 10*time.Second, "Per-request timeout")
	maxIdle := flag.Int("max-idle", 0, "Max idle connections per host (0 = use -c value)")
	var headers headerFlags
	flag.Var(&headers, "header", "Custom header in 'Key: Value' format (repeatable)")
	flag.Parse()

	if *url == "" {
		fmt.Fprintln(os.Stderr, "error: -url is required")
		flag.Usage()
		os.Exit(1)
	}
	if *c < 1 {
		fmt.Fprintln(os.Stderr, "error: -c must be >= 1")
		os.Exit(1)
	}

	idlePerHost := *c
	if *maxIdle > 0 {
		idlePerHost = *maxIdle
	}

	client := &http.Client{
		Timeout: *timeout,
		Transport: &http.Transport{
			MaxIdleConns:        idlePerHost,
			MaxIdleConnsPerHost: idlePerHost,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\nReceived interrupt, shutting down...")
		cancel()
	}()

	workCh := make(chan struct{}, *c)
	resultCh := make(chan Result, *c)

	var wg sync.WaitGroup

	// Collector
	var results []Result
	collectorDone := make(chan struct{})
	go func() {
		for r := range resultCh {
			results = append(results, r)
		}
		close(collectorDone)
	}()

	// Workers
	for i := 0; i < *c; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range workCh {
				reqCtx, reqCancel := context.WithTimeout(ctx, *timeout)
				req, err := http.NewRequestWithContext(reqCtx, *method, *url, nil)
				if err != nil {
					resultCh <- Result{Error: err, Timestamp: time.Now()}
					reqCancel()
					continue
				}
				for _, h := range headers {
					parts := strings.SplitN(h, ":", 2)
					if len(parts) == 2 {
						req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
					}
				}

				start := time.Now()
				resp, err := client.Do(req)
				latency := time.Since(start)

				if err != nil {
					resultCh <- Result{Error: err, Latency: latency, Timestamp: start}
					reqCancel()
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				resultCh <- Result{
					StatusCode: resp.StatusCode,
					Latency:    latency,
					Timestamp:  start,
				}
				reqCancel()
			}
		}()
	}

	// Producer
	startTime := time.Now()
	useDuration := *duration > 0

	if *rate > 0 {
		ticker := time.NewTicker(time.Second / time.Duration(*rate))
		defer ticker.Stop()

		sent := 0
		for {
			select {
			case <-ctx.Done():
				goto producerDone
			case <-ticker.C:
				if !useDuration && sent >= *n {
					goto producerDone
				}
				if useDuration && time.Since(startTime) >= *duration {
					goto producerDone
				}
				select {
				case workCh <- struct{}{}:
					sent++
				case <-ctx.Done():
					goto producerDone
				}
			}
		}
	} else {
		sent := 0
		for {
			if !useDuration && sent >= *n {
				break
			}
			if useDuration && time.Since(startTime) >= *duration {
				break
			}
			select {
			case workCh <- struct{}{}:
				sent++
			case <-ctx.Done():
				goto producerDone
			}
		}
	}

producerDone:
	close(workCh)

	wg.Wait()
	close(resultCh)
	<-collectorDone

	printSummary(results, time.Since(startTime))
}

func printSummary(results []Result, totalDuration time.Duration) {
	if len(results) == 0 {
		fmt.Println("\n--- Results ---")
		fmt.Println("No requests completed.")
		return
	}

	var successful, failed int
	statusCodes := make(map[int]int)
	var latencies []time.Duration

	for _, r := range results {
		if r.Error != nil {
			failed++
			continue
		}
		successful++
		statusCodes[r.StatusCode]++
		latencies = append(latencies, r.Latency)
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	total := len(results)
	rps := float64(total) / totalDuration.Seconds()

	fmt.Println("\n--- Results ---")
	fmt.Printf("Total requests:    %d\n", total)
	fmt.Printf("Successful:        %d\n", successful)
	fmt.Printf("Failed:            %d\n", failed)
	fmt.Printf("Total duration:    %s\n", totalDuration.Truncate(time.Millisecond))
	fmt.Printf("Requests/sec:      %.2f\n", rps)

	if len(latencies) > 0 {
		fmt.Println("\nLatency:")
		fmt.Printf("  p50:   %s\n", formatDuration(percentile(latencies, 0.50)))
		fmt.Printf("  p90:   %s\n", formatDuration(percentile(latencies, 0.90)))
		fmt.Printf("  p95:   %s\n", formatDuration(percentile(latencies, 0.95)))
		fmt.Printf("  p99:   %s\n", formatDuration(percentile(latencies, 0.99)))
		fmt.Printf("  max:   %s\n", formatDuration(latencies[len(latencies)-1]))
	}

	fmt.Println("\nStatus codes:")
	codes := make([]int, 0, len(statusCodes))
	for code := range statusCodes {
		codes = append(codes, code)
	}
	sort.Ints(codes)
	for _, code := range codes {
		fmt.Printf("  %d: %d\n", code, statusCodes[code])
	}
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(p*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.2fµs", float64(d.Microseconds()))
	}
	return fmt.Sprintf("%.2fms", float64(d.Microseconds())/1000.0)
}
