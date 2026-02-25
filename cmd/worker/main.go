package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/zaminda/mesh-runner/internal"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Detect if running in Kubernetes
	inKubernetes := os.Getenv("KUBERNETES_SERVICE_HOST") != ""
	if inKubernetes {
		log.Println("Running in Kubernetes environment")
	} else {
		log.Println("Running in non-Kubernetes environment")
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		panic(err)
	}
	defer nc.Close()

	nc.Subscribe(internal.RequestNATSSubject, func(msg *nats.Msg) {
		fmt.Println("received message:", string(msg.Data))
	})

	log.Println("worker started, waiting for jobs...")

	if inKubernetes {
		go runWithLeaderElection(ctx)
	}

	// Wait for SIGTERM or SIGINT
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("shutting down worker...")
	log.Println("worker stopped")
}

func runWithLeaderElection(ctx context.Context) {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	podName := os.Getenv("POD_NAME")
	if podName == "" {
		podName = "mesh-runner-worker-" + os.Getenv("HOSTNAME")
	}

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      "mesh-runner-worker-leader",
			Namespace: namespace,
		},
		Client: clientset.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: podName,
		},
	}

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:          lock,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				log.Println("I AM THE LEADER")
			},
			OnStoppedLeading: func() {
				log.Println("I AM NO LONGER THE LEADER")
			},
			OnNewLeader: func(identity string) {
				if identity != podName {
					log.Printf("New leader elected: %s\n", identity)
				}
			},
		},
	})
}
