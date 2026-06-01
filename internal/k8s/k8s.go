package k8s

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	v2 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
)

type Layer struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
}

func New(configPath string) (*Layer, error) {
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return nil, fmt.Errorf("k8s layer: failed to build config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("k8s layer: failed to create clientset: %w", err)
	}

	return &Layer{
		clientset: clientset,
		config:    config,
	}, nil
}

func (s *Layer) GetService(ctx context.Context, namespace string, serviceName string) (*v2.Service, error) {
	service, err := s.
		clientset.
		CoreV1().
		Services(namespace).
		Get(ctx, serviceName, v1.GetOptions{})

	if err != nil {
		return nil, fmt.Errorf("k8s layer: %w", err)
	}

	return service, nil
}

func (s *Layer) GetPods(ctx context.Context, srv *v2.Service) ([]v2.Pod, error) {
	pods, err := s.
		clientset.
		CoreV1().
		Pods(srv.Namespace).
		List(
			ctx,
			v1.ListOptions{
				LabelSelector: v1.FormatLabelSelector(v1.SetAsLabelSelector(srv.Spec.Selector)),
			})
	if err != nil {
		return nil, fmt.Errorf("k8s layer: failed to list pods for service %s in namespace %s: %w", srv.Name, srv.Namespace, err)
	}

	return pods.Items, nil
}

func (s *Layer) Forward(ctx context.Context, req PortForwardRequest) error {
	pod, err := s.getAPodFromService(ctx, req.Service)
	if err != nil {
		return fmt.Errorf("k8s layer: failed to get pod for port forwarding: %w", err)
	}

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward/", pod.Namespace, pod.Name)
	hostIP := strings.TrimLeft(s.config.Host, "htps:/")

	u, err := constructURL("https", hostIP, path)
	if err != nil {
		return fmt.Errorf("k8s layer: error for url: %w", err)
	}

	dialer, err := portforward.NewSPDYOverWebsocketDialer(u, s.config)
	if err != nil {
		return fmt.Errorf("k8s layer: failed to create dialer for port forwarding: %w", err)
	}
	fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", req.LocalPort, req.RemotePort)}, req.StopCh, req.ReadyCh, nil, os.Stderr)
	if err != nil {
		return fmt.Errorf("k8s layer: failed to create port forwarder: %w", err)
	}

	if err := fw.ForwardPorts(); err != nil {
		return fmt.Errorf("k8s layer: failed to forward ports: %w", err)
	}

	return nil
}

func (s *Layer) getAPodFromService(ctx context.Context, srv *v2.Service) (*v2.Pod, error) {
	pods, err := s.GetPods(ctx, srv)
	if err != nil {
		return nil, err
	}
	if len(pods) == 0 {
		return nil, fmt.Errorf("no pods found for service %s", srv.Name)
	}

	for _, p := range pods {
		if p.Status.Phase == "Running" {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("no running pods found for service %s", srv.Name)
}

func (s *Layer) ReadPodLogs(ctx context.Context, namespace string, podName string) (io.ReadCloser, error) {
	req := s.clientset.CoreV1().Pods(namespace).GetLogs(podName, &v2.PodLogOptions{
		Follow:    true,
		SinceTime: &v1.Time{Time: time.Now()},
	})
	logStream, err := req.Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("k8s: error opening log stream for pod %s: %w", podName, err)
	}

	return logStream, nil
}

func (s *Layer) GetServiceLogs(ctx context.Context, namespace string, serviceName string) (map[string]io.ReadCloser, error) {
	service, err := s.GetService(ctx, namespace, serviceName)
	if err != nil {
		return nil, err
	}

	pods, err := s.GetPods(ctx, service)
	if err != nil {
		return nil, err
	}

	if len(pods) == 0 {
		return nil, fmt.Errorf("no pods found for service %s in namespace %s", serviceName, namespace)
	}

	streams := make(map[string]io.ReadCloser)
	for _, pod := range pods {
		// Skip pods that are not running
		if pod.Status.Phase != "Running" {
			continue
		}
		req := s.clientset.CoreV1().Pods(namespace).GetLogs(pod.Name, &v2.PodLogOptions{
			Follow:    true,
			SinceTime: &v1.Time{Time: time.Now()},
		})
		logStream, err := req.Stream(ctx)
		if err != nil {
			return nil, fmt.Errorf("k8s: error opening log stream for pod %s: %w", pod.Name, err)
		}

		streams[pod.Name] = logStream
	}

	return streams, nil
}

// constructURL constructs a URL from the given scheme, host, and path.
// it is used to properly construct an URL object when the 'host' part has
// already part of the 'path' for some reason.
func constructURL(scheme, host, path string) (*url.URL, error) {
	if strings.Contains(host, "/") {
		basePath := host[strings.Index(host, "/"):]
		host = host[:len(host)-len(basePath)]
		path = basePath + path
	}

	pathEsc, err := url.QueryUnescape(path)
	if err != nil {
		return nil, err
	}

	return &url.URL{Scheme: scheme, Path: pathEsc, Host: host}, nil
}
