// log_viewer/main.go

package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Utility function to get environment variable with a fallback value
func getEnvWithFallback(envVarName, defaultValue string) string {
	val := os.Getenv(envVarName)
	if val == "" {
		return defaultValue
	}
	return val
}

func detectInput() ([]string, bool, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, false, fmt.Errorf("error checking stdin: %v", err)
	}

	// Check if there's piped input
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return nil, false, fmt.Errorf("error reading stdin: %v", err)
		}
		log.Println("Stdin detected with input:", lines)
		return lines, true, nil
	}
	log.Println("No stdin detected")
	return nil, false, nil
}

func getInputSource(envVarName, defaultValue string) ([]string, error) {
	lines, stdinDetected, err := detectInput()
	if err != nil {
		return nil, err
	}

	if stdinDetected {
		return lines, nil
	}

	envValue := getEnvWithFallback(envVarName, defaultValue)
	return []string{envValue}, nil
}

// CreateKubeClient initializes a Kubernetes client, supporting both in-cluster and local kubeconfig setups.
func CreateKubeClient() (*kubernetes.Clientset, error) {
	// Try in-cluster configuration first
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fall back to local kubeconfig
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			home := os.Getenv("HOME")
			kubeconfig = fmt.Sprintf("%s/.kube/config", home)
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig: %v", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %v", err)
	}
	return clientset, nil
}

// FetchLogsFromK8s retrieves logs for a specific pod and container from Kubernetes.
func FetchLogsFromK8s(clientset *kubernetes.Clientset, namespace, podName, containerName string) ([]string, error) {
	podLogOptions := &v1.PodLogOptions{
		Container: containerName,
	}

	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, podLogOptions)
	logStream, err := req.Stream(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("error streaming logs from pod: %v", err)
	}
	defer logStream.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(logStream)
	if err != nil {
		return nil, fmt.Errorf("error reading log stream: %v", err)
	}

	// Split logs into lines
	logLines := bytes.Split(buf.Bytes(), []byte("\n"))
	var logs []string
	for _, line := range logLines {
		logs = append(logs, string(line))
	}

	return logs, nil
}

func main() {
	// Check for stdin input first
	rawLogs, err := getInputSource("MY_ENV_VAR", "default_value")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		log.Println("Error getting input source:", err)
		os.Exit(1)
	}

	// If no stdin input is detected, check for Kubernetes environment variables
	if len(rawLogs) == 0 {
		podName := os.Getenv("PLUGIN_POD")
		namespace := os.Getenv("PLUGIN_NAMESPACE")
		containerName := os.Getenv("PLUGIN_CONTAINER")

		if podName != "" && namespace != "" && containerName != "" {
			log.Println("Using Kubernetes mode with pod:", podName, "namespace:", namespace, "container:", containerName)
			clientset, err := CreateKubeClient()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating Kubernetes client: %v\n", err)
				log.Println("Error creating Kubernetes client:", err)
				os.Exit(1)
			}

			rawLogs, err = FetchLogsFromK8s(clientset, namespace, podName, containerName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching logs: %v\n", err)
				log.Println("Error fetching logs:", err)
				os.Exit(1)
			}
		} else {
			log.Println("No Kubernetes environment variables set and no stdin input detected")
			fmt.Fprintf(os.Stderr, "No input source detected\n")
			os.Exit(1)
		}
	}

	log.Println("Raw logs:", rawLogs)

	parsedLogs, err := parseRawLogs(rawLogs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing logs: %v\n", err)
		log.Println("Error parsing logs:", err)
		os.Exit(1)
	}

	model := Model{
		logs:         parsedLogs,
		filteredLogs: parsedLogs,
	}

	log.Println("Starting TUI with logs:", parsedLogs)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if err := p.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting TUI: %v\n", err)
		log.Println("Error starting TUI:", err)
		os.Exit(1)
	}
}
