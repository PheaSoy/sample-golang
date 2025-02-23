package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// AlertConfig defines the webhook or email alert details
type AlertConfig struct {
	WebhookURL string
	Email      string
}

var alertConfig = AlertConfig{
	WebhookURL: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL", // Replace with your webhook
	Email:      "alert@example.com",                                  // Replace with your alert email
}

// sendAlert sends an alert when a container is unhealthy
func sendAlert(containerID, containerName, status string) {
	message := fmt.Sprintf("🚨 Alert: Docker container %s (%s) is %s", containerName, containerID[:12], status)

	// Send alert to webhook (e.g., Slack, Teams)
	payload := map[string]string{"text": message}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(alertConfig.WebhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("Error sending alert: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("✅ Alert sent successfully:", message)
}

// checkDockerHealth checks the health status of running containers
func checkDockerHealth() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		log.Fatalf("Failed to list containers: %v", err)
	}

	for _, container := range containers {
		containerID := container.ID
		containerName := container.Names[0]
		containerStatus := container.Status

		// If container has health check info
		inspect, err := cli.ContainerInspect(ctx, containerID)
		if err != nil {
			log.Printf("Error inspecting container %s: %v", containerID, err)
			continue
		}

		if inspect.State.Health != nil {
			healthStatus := inspect.State.Health.Status
			fmt.Printf("Container %s (%s) is %s\n", containerName, containerID[:12], healthStatus)

			if healthStatus == "unhealthy" {
				sendAlert(containerID, containerName, healthStatus)
			}
		} else {
			fmt.Printf("Container %s (%s) has no health check defined.\n", containerName, containerID[:12])
		}
	}
}

// restartUnhealthyContainers restarts unhealthy containers
func restartUnhealthyContainers() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		log.Fatalf("Failed to list containers: %v", err)
	}

	for _, container := range containers {
		containerID := container.ID
		containerName := container.Names[0]

		inspect, err := cli.ContainerInspect(ctx, containerID)
		if err != nil {
			log.Printf("Error inspecting container %s: %v", containerID, err)
			continue
		}

		if inspect.State.Health != nil && inspect.State.Health.Status == "unhealthy" {
			fmt.Printf("Restarting container %s (%s)...\n", containerName, containerID[:12])
			err := cli.ContainerRestart(ctx, containerID, nil)
			if err != nil {
				log.Printf("Failed to restart container %s: %v", containerName, err)
			} else {
				fmt.Printf("✅ Container %s restarted successfully.\n", containerName)
			}
		}
	}
}

func main() {
	fmt.Println("🔍 Checking Docker container health...")
	checkDockerHealth()

	// Optional: Restart unhealthy containers
	// restartUnhealthyContainers()
}
