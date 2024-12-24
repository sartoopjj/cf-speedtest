package main

import (
	"fmt"
	"log"

	speedtest "github.com/sartoopjj/cloudflare-speedtest/speedtest"
)

func main() {
	result, err := speedtest.NewSpeedTester(speedtest.SpeedTestConfig{
		CloudflareIP: "162.159.140.221",
		// CloudflareIP: "", // Use DNS
		PacketSize:  100 * 1024 * 1024, // MB
		PacketCount: 5,
		Verbose:     true,
	}).RunTest()

	if err != nil {
		log.Fatalf("Speed test failed: %v", err)
	}

	fmt.Printf("Average Download Speed: %.2f Mb/s\n", result.AvgDownloadSpeedMb)
	fmt.Printf("Average Upload Speed: %.2f Mb/s\n", result.AvgUploadSpeedMb)
	fmt.Printf("Average Download Latency: %v\n", result.AvgDownloadLatency)
	fmt.Printf("Average Upload Latency: %v\n", result.AvgUploadLatency)
}
