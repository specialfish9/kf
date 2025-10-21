package k8s

import (
	"context"
	"fmt"
	v2 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"net/url"
	"os"
	"strings"
)

type Service struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
}

func NewService(configPath string) (*Service, error) {
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return nil, fmt.Errorf("k8s service: failed to build config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("k8s service: failed to create clientset: %w", err)
	}

	return &Service{
		clientset: clientset,
		config:    config,
	}, nil
}

func (s *Service) GetService(ctx context.Context, namespace string, serviceName string) (*v2.Service, error) {
	service, err := s.
		clientset.
		CoreV1().
		Services(namespace).
		Get(ctx, serviceName, v1.GetOptions{})

	if err != nil {
		return nil, fmt.Errorf("k8s service: failed to get service %s in namespace %s: %w", serviceName, namespace, err)
	}

	return service, nil
}

func (s *Service) GetPods(ctx context.Context, srv *v2.Service) ([]v2.Pod, error) {
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
		return nil, fmt.Errorf("k8s service: failed to list pods for service %s in namespace %s: %w", srv.Name, srv.Namespace, err)
	}

	return pods.Items, nil
}

func (s *Service) Forward(ctx context.Context, req PortForwardRequest) error {
	pod, err := s.getAPodFromService(ctx, req.Service)
	if err != nil {
		return fmt.Errorf("k8s service: failed to get pod for port forwarding: %w", err)
	}

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward/", pod.Namespace, pod.Name)
	hostIP := strings.TrimLeft(s.config.Host, "htps:/")

	u, err := constructURL("https", hostIP, path)
	if err != nil {
		return fmt.Errorf("k8s service: error for url: %w", err)
	}

	dialer, err := portforward.NewSPDYOverWebsocketDialer(u, s.config)
	if err != nil {
		return fmt.Errorf("k8s service: failed to create dialer for port forwarding: %w", err)
	}
	fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", req.LocalPort, req.RemotePort)}, req.StopCh, req.ReadyCh, nil, os.Stderr)
	if err != nil {
		return fmt.Errorf("k8s service: failed to create port forwarder: %w", err)
	}

	if err := fw.ForwardPorts(); err != nil {
		return fmt.Errorf("k8s service: failed to forward ports: %w", err)
	}

	return nil
}

func (s *Service) getAPodFromService(ctx context.Context, srv *v2.Service) (*v2.Pod, error) {
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
