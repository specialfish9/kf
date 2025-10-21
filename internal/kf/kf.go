package kf

import (
	"cmp"
	"context"
	"kf/config"
	"kf/internal/k8s"
	"log/slog"
)

type KF struct {
	srv *k8s.Service
}

func New(k8sConfigPath string) (*KF, error) {
	srv, err := k8s.NewService(k8sConfigPath)
	if err != nil {
		return nil, err
	}
	return &KF{srv: srv}, nil
}

func (k *KF) ForwardProfile(ctx context.Context, profile *config.Profile, namespace string, stopCh chan struct{}) {
	for _, overlay := range profile.Services {
		go func() {
			for {
				slog.Debug("kf: forwarding profile", "name", overlay.Service.Alias)
				if err := k.forwardOverlay(ctx, overlay, cmp.Or(namespace, profile.Namespace), stopCh); err != nil {
					slog.Error("kf: error forwarding profile '%s': %v", overlay.Service.Alias, err.Error())
				}
			}
		}()
	}
}

func (k *KF) ForwardOverlays(ctx context.Context, overlays []*config.ServiceOverlay, namespace string, stopCh chan struct{}) {
	for _, overlay := range overlays {
		go func() {
			for {
				slog.Debug("kf: forwarding overlay", "name", overlay.Service.Alias)
				if err := k.forwardOverlay(ctx, overlay, namespace, stopCh); err != nil {
					slog.Error("kf: error forwarding overlay '%s': %v", overlay.Service.Alias, err.Error())
				}
			}
		}()
	}
}

func (k *KF) ForwardServices(ctx context.Context, services []*config.Service, namespace string, stopCh chan struct{}) {
	for _, service := range services {
		go func() {
			for {
				slog.Debug("kf: forwarding service", "name", service.Alias)
				if err := k.forwardService(ctx, service, namespace, stopCh); err != nil {
					slog.Error("kf: error forwarding service '%s': %v", service.Alias, err.Error())
				}
			}
		}()
	}
}

func (k *KF) forwardOverlay(ctx context.Context, overlay *config.ServiceOverlay, namespace string, stopCh chan struct{}) error {
	return k.forwardService(
		ctx,
		&config.Service{
			Name:       overlay.Service.Name,
			Alias:      overlay.Service.Alias,
			LocalPort:  cmp.Or(overlay.LocalPort, overlay.Service.LocalPort),
			RemotePort: cmp.Or(overlay.RemotePort, overlay.Service.RemotePort),
		},
		namespace,
		stopCh,
	)
}

func (k *KF) forwardService(ctx context.Context, cfgService *config.Service, namespace string, stopCh chan struct{}) error {
	slog.Info(
		"Forwarding service",
		"name", cfgService.Alias,
		"ns", namespace,
		"lport", cfgService.LocalPort,
		"rport", cfgService.RemotePort,
	)

	srv, err := k.srv.GetService(ctx, namespace, cfgService.Name)
	if err != nil {
		return err
	}

	readyCh := make(chan struct{})

	err = k.srv.Forward(
		ctx,
		k8s.PortForwardRequest{
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

	slog.Info("service stopped", "name", cfgService.Alias, "lport", cfgService.LocalPort)
	return nil
}
