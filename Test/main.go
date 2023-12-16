package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/sqweek/dialog"
)

var list = [10]string{"Albert", "Beatrice", "Corrine", "Dorian", "Etienne", "Francois", "Gertrude", "Henry", "Ines", "Janine"}
var root Tree

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
	Path string `json:"path"`
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
	peerMsg := PeerMsg{List: list[:]}
	time.Sleep(time.Second * 5)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	err := json.NewEncoder(w).Encode(peerMsg)

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

	time.Sleep(time.Second * 5)

	fmt.Println("PeerName :", m.PeerName)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	err := json.NewEncoder(w).Encode(root)
	fmt.Println(root)
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
	fmt.Println(m.Path)
	time.Sleep(time.Second * 5)
}

func main() {
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/export", handleExport)
	http.HandleFunc("/peer", handlePeer)
	http.HandleFunc("/data", handlePeerData)
	http.HandleFunc("/download", handleDownload)

	t1 := Tree{Name: "Leaf 1", Children: make([]*Tree, 0)}
	t2 := Tree{Name: "Leaf 2", Children: make([]*Tree, 0)}
	t3 := Tree{Name: "Leaf 3", Children: make([]*Tree, 0)}

	t4 := Tree{Name: "Leaf 4", Children: make([]*Tree, 0)}
	t5 := Tree{Name: "Leaf 5", Children: make([]*Tree, 0)}
	t6 := Tree{Name: "Leaf 6", Children: make([]*Tree, 0)}

	n1 := Tree{Name: "Node 1", Children: make([]*Tree, 0)}
	n2 := Tree{Name: "Node 2", Children: make([]*Tree, 0)}

	n1.Children = append(n1.Children, &t1)
	n1.Children = append(n1.Children, &t2)
	n1.Children = append(n1.Children, &t3)

	n2.Children = append(n2.Children, &t4)
	n2.Children = append(n2.Children, &t5)
	n2.Children = append(n2.Children, &t6)

	root1 := Tree{Name: "Root", Children: make([]*Tree, 0)}
	root1.Children = append(root1.Children, &n1)
	root1.Children = append(root1.Children, &n2)

	root = root1

	fmt.Println("Listening on :8080")
	http.ListenAndServe(":8080", nil)

}
