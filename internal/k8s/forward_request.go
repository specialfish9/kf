package k8s

import v2 "k8s.io/api/core/v1"

type PortForwardRequest struct {
	Service    *v2.Service
	LocalPort  int
	RemotePort int
	StopCh     <-chan struct{}
	ReadyCh    chan struct{}
}
