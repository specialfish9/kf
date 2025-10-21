# KF
A lazy tool for forwarding Kubernetes pods.

Installation:
```bash
sudo make install
```

and optionally copy the default config:
```bash
sudo make config
```

Usage:
```bash
kf -p (--profile) <profile> [-n namepspace]
kf -s (--service) <alias service>[:lport][:rport]... [-n namepspace] 
kf -f (--forward) <nome pod>:<lport>:<rport>... [-n namespace] 
kf -l (--list)
kf -h (--help)
```
