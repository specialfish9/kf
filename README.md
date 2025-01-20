# KF
A lazy tool for forwarding  services in k8

Installation:
```bash
make install
```

Usage:
```bash
kf -p (--profile) <profile> [-n namepspace]
kf -s (--service) <alias service>[:lport][:rport] [-n namepspace] 
kf -f (--forward) <nome pod>:<lport>:<rport> [-n namespace] 
kf -l (--list)
kf -h (--help)
```
