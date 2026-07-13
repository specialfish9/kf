package main

import (
	"cmp"
	"fmt"
	"kf/cmd/kl/model"
	"kf/config"
	"kf/internal/kl"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"k8s.io/client-go/util/homedir"
)

func getK8sConfigPath() string {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig != "" {
		return kubeconfig
	}
	home := homedir.HomeDir()
	return filepath.Join(home, ".kube", "config")
}

func main() {
	args := parseArgs()

	configPath := cmp.Or(*args.config, config.DefaultPath())

	cfg, err := config.Read(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kl: unable to load config: %v", err.Error())
	}

	var allMode bool
	if args.all != nil {
		allMode = *args.all
	}

	kl, err := kl.New(cfg, getK8sConfigPath(), *args.service, *args.namespace, args.filter, allMode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kl: unable to initialize kl: %v", err.Error())
	}

	m := model.New(kl)

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Ups...an error occured: %v", err)
	}

}
