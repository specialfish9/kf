package kf

import (
	"fmt"
)

func (k *KF) List() {
	fmt.Println("Profiles:")
	for _, profile := range k.cfg.Profiles {
		line := fmt.Sprintf("\t- %s", profile.Name)
		if ns, ok := profile.Namespace.Val(); ok {
			line += fmt.Sprintf(" (namespace: %s)", ns)
		}
		fmt.Println(line + ":")

		for _, overlay := range profile.Services {
			line := fmt.Sprintf("\t\t- %s", overlay.Ref)
			if ns, ok := overlay.Namespace.Val(); ok {
				line += fmt.Sprintf(" (namespace: %s)", ns)
			}
			fmt.Println(line)
		}
	}

	fmt.Println("Services:")
	for _, svc := range k.cfg.Services {
		line := fmt.Sprintf("\t- %s", svc.Name)
		if ns, ok := svc.Alias.Val(); ok {
			line += fmt.Sprintf(" (alias: %s)", ns)
		}
		fmt.Println(line)
	}
}
