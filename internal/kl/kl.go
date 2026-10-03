package kl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	configv3 "kf/config/v3"
	"kf/internal/k8s"
	"regexp"
	"time"
)

type KL struct {
	layer       *k8s.Layer
	cfg         *configv3.Config
	serviceName string
	namespace   string
	filter      *regexp.Regexp
	allMode     bool
}

func New(cfg *configv3.Config, k8sConfigPath string, serviceName string, namespace string, filter *string, allMode bool) (*KL, error) {
	var regexpFilter *regexp.Regexp
	if filter != nil {
		var err error
		regexpFilter, err = regexp.Compile(*filter)
		if err != nil {
			return nil, fmt.Errorf("kl: invalid regexp format '%s'", *filter)
		}
	}

	layer, err := k8s.New(k8sConfigPath)
	if err != nil {
		return nil, fmt.Errorf("kl: %s", err.Error())
	}

	// Support for alias
	kfService := cfg.GetService(serviceName)
	if kfService != nil {
		serviceName = kfService.Name
	}

	return &KL{
		layer:       layer,
		cfg:         cfg,
		serviceName: serviceName,
		namespace:   namespace,
		filter:      regexpFilter,
		allMode:     allMode,
	}, nil
}

func (k *KL) Service() string {
	return k.serviceName
}

func (k *KL) Namespace() string {
	return k.namespace
}

func (k *KL) Filter() string {
	if k.filter != nil {
		return k.filter.String()
	}
	return ""
}

func (k *KL) ReadLogs(ctx context.Context) (<-chan string, <-chan error, error) {
	// Get the K8s service
	service, err := k.layer.GetService(ctx, k.namespace, k.serviceName)
	if err != nil {
		return nil, nil, fmt.Errorf("kl: cannot fetch service '%s' in namespace '%s': %w", k.serviceName, k.namespace, err)
	}

	pods, err := k.layer.GetPods(ctx, service)
	if err != nil {
		return nil, nil, fmt.Errorf("kl: cannot fetch pods for service '%s' in namespace '%s': %w", k.serviceName, k.namespace, err)
	}

	outChan := make(chan string)
	errChan := make(chan error)

	showPodLabel := len(pods) > 1 // TODO this should be counted using running pod only
	for _, pod := range pods {
		// Filter non-running pods
		if pod.Status.Phase != "Running" {
			continue
		}

		go k.readPodLog(ctx, pod.Name, showPodLabel, outChan, errChan)
	}

	return outChan, errChan, nil
}

func (k *KL) readPodLog(
	ctx context.Context,
	podName string,
	showPodName bool,
	outChan chan<- string,
	errChan chan<- error,
) {
	var since time.Time
	if k.allMode {
		since = time.Time{} // Zero
	} else {
		since = time.Now()
	}

	for {
		stream, err := k.layer.ReadPodLogsSince(ctx, k.namespace, podName, since)
		if err != nil {
			errChan <- fmt.Errorf("kl: opening stream: %w", err)
		}

		defer func() {
			_ = stream.Close()
		}()

		since = time.Now()

		scanner := bufio.NewScanner(stream)
		for scanner.Scan() {
			line := scanner.Text()

			if k.filter != nil && !k.filter.Match([]byte(line)) {
				continue
			}

			// send each line to channel
			if showPodName {
				outChan <- fmt.Sprintf("[%s] %s", podName, line)
			} else {
				outChan <- line
			}

		}

		if err := scanner.Err(); err != nil && err != io.EOF {
			errChan <- fmt.Errorf("Stream disconnected. Reconnecting...")
		} else {
			errChan <- fmt.Errorf("error reading from reader: %w", err)
		}
	}
}
