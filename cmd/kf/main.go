package main

import (
	"cmp"
	"context"
	"fmt"
	"kf/config"
	configv3 "kf/config/v3"
	"kf/internal/kf"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/SladkyCitron/slogcolor"
	"github.com/fatih/color"
	"k8s.io/client-go/util/homedir"
)

const version = "2.3.0"

func getK8sConfigPath() string {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig != "" {
		return kubeconfig
	}
	home := homedir.HomeDir()
	return filepath.Join(home, ".kube", "config")
}

func setupLogging(verbose bool) {
	options := slogcolor.Options{
		SrcFileMode: slogcolor.Nop,
		LevelTags: map[slog.Level]string{
			slog.LevelDebug: color.New(color.FgHiCyan).Sprint("DEBUG"),
			slog.LevelInfo:  color.New(color.FgHiGreen).Sprint("INFO "),
			slog.LevelWarn:  color.New(color.FgHiYellow).Sprint("WARN "),
			slog.LevelError: color.New(color.FgHiRed).Sprint("ERROR"),
		},
	}

	if verbose {
		options.Level = slog.LevelDebug
	} else {
		options.Level = slog.LevelInfo
	}

	slog.SetDefault(slog.New(slogcolor.NewHandler(os.Stderr, &options)))
}

func main() {
	printFiglet(version)

	opt := parseArgs()

	setupLogging(*opt.verbose)

	if *opt.profile == "" && len(*opt.service) == 0 && !*opt.list {
		fmt.Print(opt.help)
		return
	}

	configPath := cmp.Or(*opt.config, config.DefaultPath())
	slog.Debug("Using config file: " + configPath)

	cfg, err := configv3.Load(configPath)
	if err != nil {
		slog.Error("kf: unable to load config", "error", err.Error())
		os.Exit(1)
	}

	k, err := kf.New(getK8sConfigPath(), cfg)
	if err != nil {
		slog.Error("kf: error while connecting to kubernetes: %v", err.Error())
		os.Exit(1)
	}

	stopCh := make(chan struct{}, 1)
	defer close(stopCh)

	ctx := context.Background()

	go func() {
		switch {
		case *opt.list:
			k.List()
			os.Exit(0)
		case *opt.profile != "":
			err := k.ForwardProfile(ctx, *opt.profile, stopCh)
			if err != nil {
				slog.Error(err.Error())
				os.Exit(1)
			}
		case opt.service != nil:
			err := k.ForwardService(ctx, *opt.service, cmp.Or(*opt.namespace, ""), stopCh)
			if err != nil {
				slog.Error(err.Error())
				os.Exit(1)
			}
		}
	}()

	//waiting for interrupt
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	if s := <-interrupt; true {
		slog.Debug("Received signal: " + s.String())
	}

	fmt.Println("\n\nBye :0")
}
