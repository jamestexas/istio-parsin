// log_viewer/run_logs.go

package main

import (
	"context"
	"fmt"
	"io"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// RunLogCommand fetches logs from a Kubernetes pod/container.
func RunLogCommand(podName, namespace, containerName string) ([]string, error) {
	// Create Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("error creating in-cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating Kubernetes client: %v", err)
	}

	// Fetch logs from Kubernetes API
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &v1.PodLogOptions{
		Container: containerName,
		Follow:    false, // Set to true for streaming logs
	})

	logStream, err := req.Stream(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error streaming logs: %v", err)
	}
	defer logStream.Close()

	var logs []string
	buffer := make([]byte, 2000)
	for {
		n, err := logStream.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("error reading log stream: %v", err)
		}
		if n == 0 {
			break
		}

		logLines := string(buffer[:n])
		logs = append(logs, logLines)
	}

	return logs, nil
}
