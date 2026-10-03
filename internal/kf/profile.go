package kf

import (
	"cmp"
	"context"
	"fmt"
	configv3 "kf/config/v3"
	"log/slog"

	"golang.org/x/sync/errgroup"
)

func (k *KF) ForwardProfile(ctx context.Context, profileName string, stopCh chan struct{}) error {
	slog.Debug("Forwarding profile", "name", profileName)

	profile := k.cfg.GetProfile(profileName)
	if profile == nil {
		return fmt.Errorf("kf: unknown profile '%s'", profileName)
	}

	eg, ctx := errgroup.WithContext(ctx)

	for _, overlay := range profile.Services {
		eg.Go(func() error {
			slog.Debug("Forwarding overlay", "name", overlay.Ref)
			err := k.forwardOverlay(ctx, overlay, profile.Namespace.Or(""), stopCh)
			if err != nil {
				return fmt.Errorf("error forwarding overlay '%s': %v", overlay.Ref, err)
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("error forwarding profile '%s': %v", profileName, err)
	}

	return nil
}

func (k *KF) forwardOverlay(ctx context.Context, overlay *configv3.ServiceOverlay, profileNs string, stopCh chan struct{}) error {
	svc := k.cfg.GetService(overlay.Ref)
	if svc == nil {
		return fmt.Errorf("unknown service '%s'", overlay.Ref)
	}

	// Prioritize the namespace from profile, if it exists, otherwise use the overlay namespace.
	namespace := cmp.Or(profileNs, overlay.Namespace.Or(""))
	localPort := overlay.LPort.Or(0)

	err := k.forwardService(ctx, svc, namespace, localPort, stopCh)
	if err != nil {
		return fmt.Errorf("error forwarding service '%s': %v", svc.Name, err)
	}

	return nil

}
