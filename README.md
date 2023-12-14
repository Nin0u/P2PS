# ProtInt

## Installation

### Dependences

Debian/Ubuntu
```
sudo apt-get install gcc libgl1-mesa-dev xorg-dev
```

Fedora
```
sudo dnf install gcc libXcursor-devel libXrandr-devel mesa-libGL-devel libXi-devel libXinerama-devel libXxf86vm-devel
```

ArchLinux
```
sudo pacman -S xorg-server-devel libxcursor libxrandr libxinerama libxi
```

### Compilation et Lancement

Dans le dossier main
```
go build
./main
```