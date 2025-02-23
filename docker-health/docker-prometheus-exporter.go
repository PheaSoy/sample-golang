package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Define Prometheus metrics
var (
	containerStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "docker_container_status",
			Help: "Docker container status (1=running, 0=stopped, -1=unhealthy)",
		},
		[]string{"host", "container_id"},
	)
)

// Fetch and update container health status
func updateMetrics() {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatalf("Error connecting to Docker: %v", err)
	}
	hostname, _ := os.Hostname()

	for {
		containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
		if err != nil {
			log.Printf("Error listing containers: %v", err)
			continue
		}

		for _, container := range containers {
			inspect, err := cli.ContainerInspect(context.Background(), container.ID)
			if err != nil {
				log.Printf("Error inspecting container %s: %v", container.ID, err)
				continue
			}

			status := inspect.State.Status
			var statusValue float64

			switch status {
			case "running":
				statusValue = 1
			case "exited", "stopped":
				statusValue = 0
			default:
				statusValue = -1
			}

			containerStatus.WithLabelValues(hostname, container.ID[:12]).Set(statusValue)
		}

		time.Sleep(30 * time.Second) // Update every 30s
	}
}

func main() {
	prometheus.MustRegister(containerStatus)

	// Start the metric collection in a goroutine
	go updateMetrics()

	// Expose metrics at /metrics
	http.Handle("/metrics", promhttp.Handler())

	log.Println("🚀 Prometheus Docker Exporter running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
