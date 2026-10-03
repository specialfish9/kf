package kf

import (
	configv3 "kf/config/v3"
	"kf/internal/k8s"
)

type KF struct {
	layer *k8s.Layer
	cfg   *configv3.Config
}

func New(k8sConfigPath string, kConfig *configv3.Config) (*KF, error) {
	l, err := k8s.New(k8sConfigPath)
	if err != nil {
		return nil, err
	}
	return &KF{
		layer: l,
		cfg:   kConfig,
	}, nil
}
