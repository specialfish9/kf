# K

KF is a lazy command-line tool designed to simplify **port forwarding for Kubernetes pods**. 
It allows you to quickly forward services, pods, and ports without manually typing long `kubectl` commands.

KL is another lazy command-line tool, this time designed to simplify **logs reading
of Kubernetes pods**. It allows you to quickly read and filter logs of your pods
without having to rely on long `kubectl` commands!

## Features
- Forward specific pods with local and remote ports
- Forward Kubernetes services by alias
- Read logs of Kubernetes pods
- List available forwarding profiles
- Easy configuration via `kf.yaml`
- Minimal dependencies

## Installation

To install both `kf` and `kl`, clone the repository and install:

```bash
sudo make install-all
```

Otherwise, to install `kf` only

```bash
sudo make installf
```

Vice versa, to install `kl`:

```bash
sudo make installl
```

(Optional) Copy the default configuration:

```bash
make config
```

## Usage

```bash
kf <profile> [-n namespace]
kf -s (--service) <alias service>[:lport][:rport]... [-n namespace]
kf -f (--forward) <pod_name>:<lport>:<rport>... [-n namespace]
kf -l (--list)
kf -h (--help) # use help to see other options!
```

```bash
kl <service|alias> -n namespace [-f regexpFitler]
```

### Examples

Forward a service using a profile:
```bash
kf dev-profile
```

Forward a service by alias:
```bash
kf -s my-service:8080:80 -n dev
```

Forward a specific pod:
```bash
kf -f mypod-123:3000:3000 -n staging
```

List available profiles:
```bash
kf -l
```

Read logs of the `xyz` pod, containing the word `foo`

```bash
kl xyz -n staging -f ".*foo.*"
```

## Configuration

`kf` uses a YAML configuration file (`kf.yaml`) to store profiles and service aliases. 
You can customize this file to suit your Kubernetes environment.

## Requirements

- Go (for building from source)
- Kubernetes cluster access (`kubectl` configured)
- Make

## Development

Clone the repository:
```bash
git clone https://github.com/specialfish9/kf.git
cd kf
```

Build:
```bash
make install
```

Run without installing:
```bash
go run ./cmd/kf
```

This README was AI generated and supervised by a human.
