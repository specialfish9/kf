package main

import (
	"cmp"
	"context"
	"fmt"
	"k8s.io/client-go/util/homedir"
	"kf/config"
	"kf/internal/kf"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

const version = "2.1"

func getK8sConfigPath() string {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig != "" {
		return kubeconfig
	}
	home := homedir.HomeDir()
	return filepath.Join(home, ".kube", "config")
}

func setupLogging(verbose bool) {
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	if verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	} else {
		slog.SetLogLoggerLevel(slog.LevelInfo)
	}
}

func main() {
	printFiglet(version)

	opt := parseArgs()

	setupLogging(*opt.verbose)

	cfg, err := config.Read(config.DefaultPath())
	if err != nil {
		log.Fatalf("kf: unable to load config: %v", err.Error())
	}

	if *opt.profile == "" && len(*opt.service) == 0 && len(*opt.forward) == 0 && !*opt.list {
		fmt.Print(opt.help)
		return
	}

	if *opt.list {
		cfg.PrintList()
		return
	}

	k, err := kf.New(getK8sConfigPath())
	if err != nil {
		slog.Error("kf: error while connecting to kubernetes: %v", err.Error())
		os.Exit(1)
	}

	stopCh := make(chan struct{}, 1)
	ctx := context.Background()

	if *opt.profile != "" {
		profile := cfg.GetProfile(*opt.profile)

		if profile == nil {
			slog.Error("kf: unknown profile '%s'", *opt.profile)
			os.Exit(1)
		}

		k.ForwardProfile(ctx, profile, *opt.namespace, stopCh)
	} else if len(*opt.service) > 0 {
		services := parseServiceArgs(*opt.service, false)

		overlays := Map(services, func(s *config.Service) *config.ServiceOverlay {
			ref := cfg.ServiceMap[s.Alias]
			if ref == nil {
				log.Fatalf("kf: unknown service alias '%s'", s.Alias)
			}
			return &config.ServiceOverlay{
				Ref:        s.Alias,
				Service:    ref,
				LocalPort:  s.LocalPort,
				RemotePort: s.RemotePort,
			}
		})

		k.ForwardOverlays(ctx, overlays, cmp.Or(*opt.namespace, config.DefaultEnv), stopCh)
	} else {
		services := parseServiceArgs(*opt.forward, true)
		k.ForwardServices(ctx, services, cmp.Or(*opt.namespace, config.DefaultEnv), stopCh)
	}

	//waiting for interrupt
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	if s := <-interrupt; true {
		slog.Debug("Received signal: " + s.String())
		close(stopCh)
	}

	fmt.Println("\n\nBye")
}

func Map[T any, U any](slice []T, functor func(T) U) []U {
	result := make([]U, 0, len(slice))
	for _, v := range slice {
		result = append(result, functor(v))
	}
	return result
}
