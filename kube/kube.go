package kube

import (
	"cmp"
	"kf/config"
	"log"
)

type Kube struct {
	kubeLayer *kubeLayer
}

func NewKube() (*Kube, error) {
	kl, err := newKubeLayer()
	if err != nil {
		return nil, err
	}

	return &Kube{kubeLayer: kl}, nil
}

func (k *Kube) ForwardProfile(profile *config.Profile, namespace config.Env, stopCh chan struct{}) {
	for _, overlay := range profile.Services {
		go func() {
			err := k.forwardOverlay(overlay, cmp.Or(namespace, profile.Namespace), stopCh)
			if err != nil {
				log.Fatalf("kf: error forwarding profile '%s': %v", overlay.Service.Alias, err.Error())
			}
		}()
	}
}

func (k *Kube) ForwardOverlays(overlays []*config.ServiceOverlay, namespace config.Env, stopCh chan struct{}) {
	for _, overlay := range overlays {
		go func() {
			err := k.forwardOverlay(overlay, namespace, stopCh)
			if err != nil {
				log.Fatalf("kf: error forwarding overlay '%s': %v", overlay.Service.Alias, err.Error())
			}
		}()
	}
}

func (k *Kube) ForwardServices(services []*config.Service, namespace config.Env, stopCh chan struct{}) {
	for _, service := range services {
		go func() {
			err := k.forwardService(service, namespace, stopCh)
			if err != nil {
				log.Fatalf("kf: error forwarding service '%s': %v", service.Alias, err.Error())
			}
		}()
	}
}

func (k *Kube) forwardOverlay(overlay *config.ServiceOverlay, namespace config.Env, stopCh chan struct{}) error {
	return k.forwardService(
		&config.Service{
			Name:       overlay.Service.Name,
			Alias:      overlay.Service.Alias,
			LocalPort:  cmp.Or(overlay.LocalPort, overlay.Service.LocalPort),
			RemotePort: cmp.Or(overlay.RemotePort, overlay.Service.RemotePort),
		}, namespace, stopCh)
}

func (k *Kube) forwardService(cfgService *config.Service, namespace config.Env, stopCh chan struct{}) error {
	log.Printf("Forwarding %s (%s) - lport %d ; rport %d\n", cfgService.Alias, namespace, cfgService.LocalPort, cfgService.RemotePort)
	srv, err := k.kubeLayer.getService(namespace, cfgService.Name)

	if err != nil {
		return err
	}

	readyCh := make(chan struct{})

	err = k.kubeLayer.forward(
		PortForwardRequest{
			Service:    srv,
			LocalPort:  cfgService.LocalPort,
			RemotePort: cfgService.RemotePort,
			ReadyCh:    readyCh,
			StopCh:     stopCh,
		})

	if err != nil {
		return err
	}

	<-stopCh

	log.Printf("stopped %s:%d\n", cfgService.Alias, cfgService.LocalPort)
	return nil
}
