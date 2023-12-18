package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"

	"github.com/sqweek/dialog"
)

var clientG *http.Client
var connG net.PacketConn

type Tree struct {
	Name     string  `json:"name"`
	Children []*Tree `json:"children"`
}

type ExportMsg struct {
	Path string `json:"path"`
}

type PeerMsg struct {
	List []string `json:"list"`
}

type PeerDLMsg struct {
	PeerName string `json:"peer"`
}

type DLMsg struct {
	Path     string `json:"path"`
	PeerName string `json:"peer"`
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("index.html"))
	tmpl.Execute(w, nil)
}

func handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		fmt.Println("Error on handleExport :", r.Method)
	}
	filename, _ := dialog.File().Load()
	fmt.Println(filename)
}

func handlePeer(w http.ResponseWriter, r *http.Request) {
	list, err := GetPeers(clientG)

	peerMsg := PeerMsg{List: list[:]}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	err = json.NewEncoder(w).Encode(peerMsg)

	if err != nil {
		fmt.Println("Error !")
		http.Error(w, err.Error(), 500)
		return
	}
}

func handlePeerData(w http.ResponseWriter, r *http.Request) {

	dec := json.NewDecoder(r.Body)
	var m PeerDLMsg
	for dec.More() {
		err := dec.Decode(&m)
		if err != nil {
			fmt.Println("Error decode", err.Error())
			http.Error(w, err.Error(), 500)
			return
		}
		break
	}

	fmt.Println("PeerName :", m.PeerName)
	handleGetData(clientG, connG, []string{"data", m.PeerName})

	index := FindCachedPeerByName(m.PeerName)
	cache_peers.mutex.Lock()
	root := cache_peers.list[index].Root
	cache_peers.mutex.Unlock()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	err := json.NewEncoder(w).Encode(root)
	if err != nil {
		fmt.Println("Error !", err.Error())
		http.Error(w, err.Error(), 500)
		return
	}
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	var m DLMsg
	for dec.More() {
		err := dec.Decode(&m)
		if err != nil {
			fmt.Println("Error decode", err.Error())
			http.Error(w, err.Error(), 500)
			return
		}
		break
	}
	handleGetDataDL(clientG, connG, []string{"data_dl", m.PeerName, m.Path})
}

func gui(client *http.Client, conn net.PacketConn) {
	clientG = client
	connG = conn

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/export", handleExport)
	http.HandleFunc("/peer", handlePeer)
	http.HandleFunc("/data", handlePeerData)
	http.HandleFunc("/download", handleDownload)

	//fmt.Println("Listening on :8080")
	http.ListenAndServe(":8080", nil)

}
