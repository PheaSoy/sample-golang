package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/slack-go/slack"
)

func main() {
	// Load Kubeconfig (Use in-cluster config if running inside K8s)
	kubeconfig := flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	flag.Parse()

	config, err := rest.InClusterConfig() // For in-cluster execution
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig) // For local execution
		if err != nil {
			log.Fatalf("Failed to load kubeconfig: %v", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Watch for pod status changes
	watchPods(clientset)
}

func watchPods(clientset *kubernetes.Clientset) {
	watcher, err := clientset.CoreV1().Pods("").Watch(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Fatalf("Failed to watch pods: %v", err)
	}
	log.Println("📡 Watching Kubernetes pods for changes...")

	for event := range watcher.ResultChan() {
		switch event.Type {
		case watch.Added:
			pod := event.Object.(*v1.Pod)
			log.Printf("🟢 New Pod: %s (Status: %s)\n", pod.Name, pod.Status.Phase)

		case watch.Modified:
			pod := event.Object.(*v1.Pod)
			log.Printf("🔄 Pod Updated: %s (Status: %s)\n", pod.Name, pod.Status.Phase)

			// Detect issues
			if pod.Status.Phase == "Failed" || pod.Status.Phase == "Unknown" {
				sendSlackAlert(fmt.Sprintf("⚠️ Pod %s is in %s state!", pod.Name, pod.Status.Phase))
			}

		case watch.Deleted:
			pod := event.Object.(*v1.Pod)
			log.Printf("❌ Pod Deleted: %s\n", pod.Name)
		}
	}
}

// Send an alert to Slack
func sendSlackAlert(message string) {
	slackToken := os.Getenv("SLACK_TOKEN") // Set in env vars
	channelID := "#alerts"

	api := slack.New(slackToken)
	_, _, err := api.PostMessage(channelID, slack.MsgOptionText(message, false))
	if err != nil {
		log.Printf("Failed to send Slack alert: %v", err)
	} else {
		log.Println("🚨 Slack alert sent!")
	}
}
