package main

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

type RequestDatum struct {
	P       Peer
	Path    string
	Hash    [32]byte
	TypeReq byte // 0 -> List les nom seulement, 1 -> download pour de vrai
}

type ListRequestDatum struct {
	mutex sync.Mutex
	list  []RequestDatum
}

var reqDatum ListRequestDatum = ListRequestDatum{list: make([]RequestDatum, 0)}

func buildRequestDatum(p Peer, path string, hash [32]byte, typeReq byte) RequestDatum {
	return RequestDatum{P: p, Path: path, Hash: hash, TypeReq: typeReq}
}

func clearRequestDatum() {
	reqDatum.mutex.Lock()
	reqDatum.list = make([]RequestDatum, 0) //On vide au cas où
	reqDatum.mutex.Unlock()
}

func download_list(hash [32]byte, typeFile byte, value []byte, conn net.PacketConn) {

	//Pop le premier element
	reqDatum.mutex.Lock()
	prevReq := reqDatum.list[0]
	reqDatum.list = reqDatum.list[1:]
	reqDatum.mutex.Unlock()

	if typeFile == DIRECTORY {
		fmt.Println("Directory recieved. Contents' hashes are : ")
		for i := 0; i < len(value); i += 64 {
			//Debug
			reqDatum.mutex.Lock()
			name := prevReq.Path + "/" + string(value[i:i+32])
			hash_child := value[i+32 : i+64]
			fmt.Printf("- Name = %s, Hash = %x \n", name, hash_child)

			//Add for the next getDatum()
			req := buildRequestDatum(prevReq.P, name, [32]byte(hash_child), 0)
			reqDatum.list = append([]RequestDatum{req}, reqDatum.list...)
			reqDatum.mutex.Unlock()
		}
	}

	//Add the element in the tree
	reqDatum.mutex.Lock()
	path := strings.Split(prevReq.Path, "/")
	prevReq.P.Root = add_node(prevReq.P.Root, path[1:], path[len(path)-1], [32]byte(hash), typeFile)
	root := prevReq.P.Root

	//Si c'est fini on print
	if len(reqDatum.list) == 0 {
		print_node(root)
		fmt.Println("END !!!!!")
		reqDatum.mutex.Unlock()
		return
	}

	for i := 0; i < len(reqDatum.list); i++ {
		fmt.Println(reqDatum.list[i].Path)
	}

	//Sinon on continue avec le prochain envoie
	_, err := sendGetDatum(conn, reqDatum.list[0].P.Addr, reqDatum.list[0].Hash)
	if err != nil {
		fmt.Println("Error sendGetDatum in download_list : ", err.Error())
		reqDatum.mutex.Unlock()
		clearRequestDatum()
		return
	}

	reqDatum.mutex.Unlock()
}

// TODO: store in tree peers the data !
func download_dl(hash [32]byte, typeFile byte, value []byte, conn net.PacketConn) {
	//Pop le premier element
	reqDatum.mutex.Lock()
	prevReq := reqDatum.list[0]
	reqDatum.list = reqDatum.list[1:]
	reqDatum.mutex.Unlock()

	if typeFile == DIRECTORY {
		//On créer le dossier
		pa := prevReq.Path
		fmt.Println("Création du dossier :", pa)
		fmt.Printf("%s|\n", pa)
		println(len(pa))
		for i := 0; i < len(pa); i++ {
			print(pa[i], " ")
		}
		err := os.MkdirAll(pa, 0777)
		if err != nil {
			fmt.Println("Error mkdir all in download_dl :", err.Error())
			clearRequestDatum()
			return
		}

		fmt.Println("Directory recieved. Contents' hashes are : ")
		for i := len(value) - 64; i >= 0; i -= 64 {
			reqDatum.mutex.Lock()
			name_byte := bytes.TrimRight(value[i:i+32], string(byte(0)))
			name := prevReq.Path + "/" + string(name_byte)
			hash_child := value[i+32 : i+64]
			fmt.Printf("- Name = %s, Hash = %x \n", name, hash_child)

			//Add for the next getDatum()
			req := buildRequestDatum(prevReq.P, name, [32]byte(hash_child), 1)
			reqDatum.list = append([]RequestDatum{req}, reqDatum.list...)
			reqDatum.mutex.Unlock()
		}
	} else if typeFile == TREE {
		fmt.Println("BigFile recieved. Contents' hashes are : ")
		for i := len(value) - 32; i >= 0; i -= 32 {
			hash_child := value[i : i+32]
			fmt.Printf("- Hash = %x\n", hash_child)
			req := buildRequestDatum(prevReq.P, prevReq.Path, [32]byte(hash_child), 1)
			reqDatum.mutex.Lock()
			reqDatum.list = append([]RequestDatum{req}, reqDatum.list...)
			reqDatum.mutex.Unlock()
		}
	} else if typeFile == CHUNK {
		file, err := os.OpenFile(prevReq.Path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.ModePerm)
		if err != nil {
			fmt.Println("Error os.OpenFile : ", err.Error())
			clearRequestDatum()
			return
		}

		file.Write(value)
		file.Close()
	} else {
		fmt.Println("Unknown type file !")
		clearRequestDatum()
		return
	}

	reqDatum.mutex.Lock()
	//Si c'est fini on print
	if len(reqDatum.list) == 0 {
		fmt.Println("END !!!!!")
		reqDatum.mutex.Unlock()
		return
	}

	for i := 0; i < len(reqDatum.list); i++ {
		fmt.Println(reqDatum.list[i].Path, " : ", reqDatum.list[i].Hash)
	}

	//Sinon on continue avec le prochain envoie
	_, err := sendGetDatum(conn, reqDatum.list[0].P.Addr, reqDatum.list[0].Hash)
	if err != nil {
		fmt.Println("Error sendGetDatum in download_dl : ", err.Error())
		reqDatum.mutex.Unlock()
		clearRequestDatum()
		return
	}

	reqDatum.mutex.Unlock()

}
