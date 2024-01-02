package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"

	"github.com/ncruces/zenity"
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
	path, err := zenity.SelectFile(zenity.Filename("~"), zenity.Directory())
	if err == nil {
		fmt.Println(path)
		execExport(connG, []string{"export", path})
	}
}

func handlePeer(w http.ResponseWriter, r *http.Request) {
	list, err := GetPeers(clientG)
	if err != nil {
		fmt.Println("Error GetPeers !")
		http.Error(w, err.Error(), 500)
		return
	}

	peerMsg := PeerMsg{List: list[:]}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	err = json.NewEncoder(w).Encode(peerMsg)

	if err != nil {
		fmt.Println("Error encode !")
		http.Error(w, err.Error(), 500)
		return
	}
}

func handlePeerData(w http.ResponseWriter, r *http.Request) {

	dec := json.NewDecoder(r.Body)
	var m PeerDLMsg
	if dec.More() {
		err := dec.Decode(&m)
		if err != nil {
			fmt.Println("Error decode", err.Error())
			http.Error(w, err.Error(), 500)
			return
		}
	}

	fmt.Println("PeerName :", m.PeerName)
	p, err := execGetData(clientG, connG, []string{"data", m.PeerName})
	if err != nil {
		fmt.Println("Error data !", err.Error())
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	err = json.NewEncoder(w).Encode(p.Root)
	if err != nil {
		fmt.Println("Error Encode !", err.Error())
		http.Error(w, err.Error(), 500)
		return
	}
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	var m DLMsg
	if dec.More() {
		err := dec.Decode(&m)
		if err != nil {
			fmt.Println("Error decode", err.Error())
			http.Error(w, err.Error(), 500)
			return
		}
	}
	fmt.Println("PATH", m.Path)
	path, _ := zenity.SelectFile(zenity.Filename("~"), zenity.Directory())
	fmt.Println(path)
	err := execGetDataDL(clientG, connG, []string{"data_dl", m.PeerName, m.Path}, path)

	if err != nil {
		fmt.Println("Error DL", err.Error())
		http.Error(w, err.Error(), 500)
		return
	}
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
