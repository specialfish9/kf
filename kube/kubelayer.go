package kube

import (
	"context"
	"fmt"
	v2 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/client-go/util/homedir"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type kubeLayer struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
}

type PortForwardRequest struct {
	Service    *v2.Service
	LocalPort  int
	RemotePort int
	StopCh     <-chan struct{}
	ReadyCh    chan struct{}
}

func newKubeLayer() (*kubeLayer, error) {
	var kubeconfig string
	
	// Check for KUBECONFIG environment variable first
	if kubeconfigEnv := os.Getenv("KUBECONFIG"); kubeconfigEnv != "" {
		kubeconfig = kubeconfigEnv
	} else {
		// Fall back to default location
		home := homedir.HomeDir()
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)

	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		return nil, err
	}

	return &kubeLayer{
		clientset: clientset,
		config:    config,
	}, nil
}

func (kl *kubeLayer) getService(namespace string, serviceName string) (*v2.Service, error) {
	service, err := kl.
		clientset.
		CoreV1().
		Services(namespace).
		Get(context.TODO(), serviceName, v1.GetOptions{})

	if err != nil {
		return nil, err
	}

	return service, nil
}

func (kl *kubeLayer) getPods(srv *v2.Service) ([]v2.Pod, error) {
	pods, err := kl.
		clientset.
		CoreV1().
		Pods(srv.Namespace).
		List(
			context.TODO(),
			v1.ListOptions{
				LabelSelector: v1.FormatLabelSelector(v1.SetAsLabelSelector(srv.Spec.Selector)),
			})
	if err != nil {
		return nil, err
	}

	return pods.Items, nil
}

func (kl *kubeLayer) forward(req PortForwardRequest) error {
	pods, err := kl.getPods(req.Service)

	if err != nil {
		return err
	}
	if len(pods) == 0 {
		return fmt.Errorf("no pods found for service %s", req.Service.Name)
	}

	var pod *v2.Pod
	for _, p := range pods {
		if p.Status.Phase == "Running" {
			pod = &p
			break
		}
	}
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward/", pod.Namespace, pod.Name)
	hostIP := strings.TrimLeft(kl.config.Host, "htps:/")

	transport, upgrader, err := spdy.RoundTripperFor(kl.config)
	if err != nil {
		return err
	}

	url, err := constructUrl("https", hostIP, path)
	if err != nil {
		return fmt.Errorf("kf: error parsing url: %w", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, url)

	fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", req.LocalPort, req.RemotePort)}, req.StopCh, req.ReadyCh, os.Stdout, os.Stderr)
	if err != nil {
		return err
	}

	return fw.ForwardPorts()
}

func constructUrl(scheme, host, path string) (*url.URL, error) {
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
