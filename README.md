# P2PS
(Stands for P2PShare, but the name has already been taken unfortunately)
UDP File sharing program using a control server.

## Features
- Login to server
- Keepalives for the server
- Retrievial of peers and their exported trees via the server
- Multithreaded download
- Asynchronous Exportation of trees
- CLI & GUI
- Cryptographic signatures : The message are signed in the initial handshake between peers in order to authenticate them
- NAT Traversal
- RTO computation and Packet Loss Handling
- Peers and Data Caching
  
## Prerequisite
- [Go](https://go.dev/)
- A Web Browser for the GUI

## How to use
- Clone the project
```
  git clone https://github.com/Nin0u/P2PS.git
```
- Inside `main/`, open a terminal and type `go build`
- You can now execute the program by typing
```
./main --username="your_username" [--gui] [--export="path_to_directory"]
```
