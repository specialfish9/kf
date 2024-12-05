package main

import (
	"fmt"
	"kf/config"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func forward(kl *kubeLayer, cfgService *config.Service, stopCh chan struct{}) error {
	log.Printf("Forwarding %s (%s) - lport %d ; rport %d\n", cfgService.Name, cfgService.Namespace, cfgService.LocalPort, cfgService.RemotePort)
	srv, err := kl.getService(cfgService.Namespace, cfgService.Name)

	if err != nil {
		return err
	}

	readyCh := make(chan struct{})

	err = kl.forward(
		PortForwardRequest{
			Service:    srv,
			LocalPort:  cfgService.LocalPort,
			RemotePort: cfgService.RemotePort,
			ReadyCh:    readyCh,
			StopCh:     stopCh,
		})

	if err != nil {
		panic(err.Error())
	}

	<-stopCh

	log.Printf("stopped %s:%d\n", cfgService.Name, cfgService.LocalPort)
	return nil
}

func printFiglet() {
	fmt.Println(" __      _____ ")
	fmt.Println("|  | ___/ ____\\ ")
	fmt.Println("|  |/ /\\   __\\ ")
	fmt.Println("|    <  |  |   ")
	fmt.Println("|__|_ \\ |__|   ")
	fmt.Println("     \\/       2")
	fmt.Println("")
}

func usage() {
	fmt.Println(os.Args[0] + " <profile name> [profile name....]")
}

func main() {
	printFiglet()

	configPath := config.DefaultPath()
	cfg, err := config.Read(configPath)

	if err != nil {
		log.Fatalf("kf: unable to load config: %v", err.Error())
	}

	kubeLayer, err := newKubeLayer()

	if err != nil {
		log.Fatalf("kf: error while connecting to kubernetes: %v", err.Error())
	}

	stopCh := make(chan struct{}, 1)

	if len(os.Args) == 1 {
		usage()
		return
	}

	for i, arg := range os.Args {
		// skip name
		if i == 0 {
			continue
		}

		var profile *config.Profile
		for _, p := range cfg.Profiles {
			if p.Name == arg {
				profile = p
				break
			}
		}

		if profile == nil {
			log.Fatalf("kf: error spinning up services: unknown profile '%s'", arg)
		}

		for _, service := range profile.Services {
			go func() {
				err := forward(kubeLayer, service, stopCh)
				if err != nil {
					log.Fatalf("kf: error forwarding service '%s': %v", service.Name, err.Error())
				}
			}()
		}

	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	if s := <-interrupt; true {
		log.Println("Signal: " + s.String())
		close(stopCh)
	}

	log.Println("Bye")
}
