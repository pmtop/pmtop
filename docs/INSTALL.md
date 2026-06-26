# Installing pmtop

## Quick Install (Linux amd64 / arm64)

```bash
# latest stable
curl -sL https://github.com/pmtop/pmtop/releases/latest/download/pmtop_Linux_x86_64.tar.gz \
  | tar -xz pmtop && sudo mv pmtop /usr/local/bin/pmtop

# verify
pmtop version
```

## Package Managers

### Debian / Ubuntu

```bash
curl -sL https://github.com/pmtop/pmtop/releases/latest/download/pmtop_amd64.deb \
  -o /tmp/pmtop.deb && sudo dpkg -i /tmp/pmtop.deb
```

### Fedora / RHEL

```bash
sudo dnf install -y https://github.com/pmtop/pmtop/releases/latest/download/pmtop_x86_64.rpm
```

### Arch Linux (AUR)

```bash
# binary package (recommended)
yay -S pmtop-bin

# source package
yay -S pmtop
```

### Homebrew (Linux)

```bash
brew install pmtop/tap/pmtop
```

## Build from Source

Requires Go 1.22+.

```bash
git clone https://github.com/pmtop/pmtop.git
cd pmtop
make build
sudo cp build/pmtop-linux-amd64 /usr/local/bin/pmtop
```

## Shell Completions

```bash
# bash
pmtop completion bash | sudo tee /etc/bash_completion.d/pmtop

# zsh
pmtop completion zsh | sudo tee /usr/share/zsh/site-functions/_pmtop

# fish
pmtop completion fish | sudo tee /etc/fish/completions/pmtop.fish
```

## Man Pages

Man pages are bundled with .deb / .rpm packages. For static binaries:

```bash
pmtop man --output-dir /usr/local/share/man
```
