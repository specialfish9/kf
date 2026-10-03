package kf

import (
	"cmp"
	"context"
	"fmt"
	configv3 "kf/config/v3"
	"kf/internal/k8s"
	"log/slog"
	"math"
	"math/rand"
)

func (k *KF) ForwardService(ctx context.Context, nameOrAlias string, namespace string, stopCh chan struct{}) error {
	cfgSvc := k.cfg.GetService(nameOrAlias)
	if cfgSvc == nil {
		return fmt.Errorf("kf: unknown service '%s'", nameOrAlias)
	}

	err := k.forwardService(ctx, cfgSvc, namespace, 0, stopCh)
	if err != nil {
		return fmt.Errorf("kf: error forwarding service '%s': %v", cfgSvc.Name, err)
	}

	return nil
}

func (k *KF) forwardService(ctx context.Context, cfgSvc *configv3.Service, namespace string, localPort int, stopCh chan struct{}) error {
	localPort = cmp.Or(localPort, cfgSvc.LPort.Or(randomPort()))
	namespace = cmp.Or(namespace, "default") // Fallback to "default" namespace if not provided

	slog.Info(
		"Forwarding service",
		"name", cfgSvc.Name,
		"ns", namespace,
		"localPort", localPort,
		"remotePort", cfgSvc.RPort,
	)

	k8sSvc, err := k.layer.GetService(ctx, namespace, cfgSvc.Name)
	if err != nil {
		return err
	}

	readyCh := make(chan struct{})

	err = k.layer.Forward(
		ctx,
		k8s.PortForwardRequest{
			Service:    k8sSvc,
			LocalPort:  localPort,
			RemotePort: cfgSvc.RPort,
			ReadyCh:    readyCh,
			StopCh:     stopCh,
		})
	if err != nil {
		return err
	}

	<-stopCh

	slog.Info("Service stopped", "name", cfgSvc.Alias, "lport", localPort)
	return nil
}

func randomPort() int {
	return rand.Intn(math.MaxUint16-1024) + 1024
}
