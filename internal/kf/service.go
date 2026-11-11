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
	"time"
)

const retryBackoff = time.Second

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

	if err := ctx.Err(); err != nil {
		return err
	}

	for {
		select {
		case <-stopCh:
			slog.Info("Received a stop command. Terminating...", "name", cfgSvc.Alias, "lport", localPort)
			return nil
		case <-ctx.Done():
			slog.Info("Context is done. Terminating...", "name", cfgSvc.Alias, "lport", localPort)
			return nil
		default:
			// Fresh channel per attempt: client-go closes ReadyCh on success,
			// reusing it across retries panics with "close of closed channel".
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
				slog.Error("Error forwarding service", "error", err, "name", cfgSvc.Alias, "lport", localPort)
			}

			// Backoff before retrying to avoid a hot loop.
			select {
			case <-stopCh:
			case <-ctx.Done():
			case <-time.After(retryBackoff):
			}
		}
	}
}

func randomPort() int {
	return rand.Intn(math.MaxUint16-1024) + 1024
}
