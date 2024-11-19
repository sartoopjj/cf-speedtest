package speedtest

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// SpeedTestConfig contains the configuration for the speed test.
type SpeedTestConfig struct {
	CloudflareIP string // Specific IP of the Cloudflare server
	PacketSize   int    // Size of each packet (in bytes)
	PacketCount  int    // Number of packets to send
	Verbose      bool
}

// SpeedTestResult holds the result of the speed test.
type SpeedTestResult struct {
	AvgUploadSpeedMb   float64
	AvgDownloadSpeedMb float64
	AvgUploadLatency   time.Duration
	AvgDownloadLatency time.Duration
}

// SpeedTester struct that will perform the speed test.
type SpeedTester struct {
	config SpeedTestConfig
	client *http.Client
	testID int64
}

// NewSpeedTester creates a new SpeedTester with a specific configuration.
func NewSpeedTester(config SpeedTestConfig) *SpeedTester {
	if config.CloudflareIP != "" {
		return &SpeedTester{
			config: config,
			client: &http.Client{
				Transport: &http.Transport{
					DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
						dialer := &net.Dialer{
							Timeout: 10 * time.Second,
						}
						return dialer.DialContext(ctx, network, net.JoinHostPort(config.CloudflareIP, "443"))
					},
				},
			},
			testID: rand.Int63(),
		}
	} else {
		return &SpeedTester{
			config: config,
			client: &http.Client{},
			testID: rand.Int63(),
		}
	}
}

// RunTest performs the upload and download speed test and returns the result.
func (st *SpeedTester) RunTest() (*SpeedTestResult, error) {
	uploadSpeed, uploadLatency, err := st.testUpload()
	if err != nil {
		return nil, fmt.Errorf("upload test failed: %w", err)
	}

	downloadSpeed, downloadLatency, err := st.testDownload()
	if err != nil {
		return nil, fmt.Errorf("download test failed: %w", err)
	}

	result := &SpeedTestResult{
		AvgUploadSpeedMb:   uploadSpeed,
		AvgDownloadSpeedMb: downloadSpeed,
		AvgUploadLatency:   uploadLatency,
		AvgDownloadLatency: downloadLatency,
	}

	return result, nil
}

func (st *SpeedTester) testUpload() (float64, time.Duration, error) {
	uploadURL := fmt.Sprintf("https://speed.cloudflare.com/__up?measId=%d", st.testID)
	rawBody := bytes.Repeat([]byte{0x30}, st.config.PacketSize)

	var totalLatency time.Duration

	for i := 0; i < st.config.PacketCount; i++ {

		req, err := http.NewRequest("POST", uploadURL, strings.NewReader(string(rawBody)))
		req.Header.Add("Content-Type", "text/plain;charset=UTF-8")

		if err != nil {
			return 0, 0, fmt.Errorf("failed to create upload request: %w", err)
		}

		startTime := time.Now()

		resp, err := st.client.Do(req)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to perform upload request: %w", err)
		}

		latency := time.Since(startTime)

		totalLatency += (latency - st.getServerTiming(&resp.Header))

		if st.config.Verbose {
			fmt.Printf("Upload %d bytes in %s (server time: %s)\n", st.config.PacketSize, latency, st.getServerTiming(&resp.Header))
		}
	}
	speedMb := (float64(st.config.PacketSize*st.config.PacketCount) / totalLatency.Seconds()) / float64(125000)
	avgLatency := totalLatency / time.Duration(st.config.PacketCount)

	return speedMb, avgLatency, nil
}

func (st *SpeedTester) testDownload() (float64, time.Duration, error) {

	downloadURL := fmt.Sprintf("https://speed.cloudflare.com/__down?measId=%d&bytes=%d", st.testID, st.config.PacketSize)

	var totalLatency time.Duration
	for i := 0; i < st.config.PacketCount; i++ {

		req, err := http.NewRequest("GET", downloadURL, nil)

		if err != nil {
			return 0, 0, fmt.Errorf("failed to create download request: %w", err)
		}

		startTime := time.Now()

		resp, err := st.client.Do(req)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to perform download request: %w", err)
		}

		latency := time.Since(startTime)

		totalLatency += (latency - st.getServerTiming(&resp.Header))

		if st.config.Verbose {
			fmt.Printf("Download %d bytes in %s (server time: %s)\n", st.config.PacketSize, latency, st.getServerTiming(&resp.Header))
		}
	}

	speedMb := (float64(st.config.PacketSize*st.config.PacketCount) / totalLatency.Seconds()) / float64(125000)
	avgLatency := totalLatency / time.Duration(st.config.PacketCount)

	return speedMb, avgLatency, nil
}

// getServerTiming Extract server time from response headers
func (st *SpeedTester) getServerTiming(headers *http.Header) time.Duration {
	i, _ := strconv.ParseFloat(strings.Split(headers.Get("Server-Timing"), "=")[1], 32)
	return time.Duration(math.Round(i)) * time.Millisecond
}
