package main

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

// Keys are Hash, Value are the value in the getDatum request
type DatumCache struct {
	content map[[32]byte][]byte
	mutex   sync.Mutex
}

var datumCache DatumCache = DatumCache{content: make(map[[32]byte][]byte)}

// TODO: mettre un timeout avec time.AfterFunc pour vider le cache
// TODO: verifier que le cache est pas trop gros
func AddDatumCache(hash [32]byte, value []byte) {
	datumCache.mutex.Lock()
	datumCache.content[hash] = value
	datumCache.mutex.Unlock()
}

func GetDatumCache(hash [32]byte) ([]byte, bool) {
	datumCache.mutex.Lock()
	value, ok := datumCache.content[hash]
	datumCache.mutex.Unlock()
	return value, ok
}

func PrintDatumCache() {
	println("Cache :")
	for k, v := range datumCache.content {
		fmt.Println(k, v)
	}
}

type RequestDatum struct {
	Path  string
	Hash  [32]byte
	Count int64
}

func buildRequestDatum(path string, hash [32]byte, count int64) RequestDatum {
	return RequestDatum{Path: path, Hash: hash, Count: count}
}

func RecupDatum(conn net.PacketConn, req *RequestDatum, p *Peer) []byte {
	//Recup the Datum
	value, ok := GetDatumCache(req.Hash)

	for j := 0; !ok && j < 5; j++ {
		fmt.Println("Send GET DATUM : " + req.Path)
		_, err := sendGetDatum(conn, p.Addr, req.Hash)

		if err != nil {
			fmt.Println("[RecupDatum] Error send getDatum", err.Error())
			return nil
		}

		value, ok = GetDatumCache(req.Hash)
	}

	if !ok {
		fmt.Println("[RecupDatum] Error on get datum " + req.Path)
		return nil
	}

	if value == nil {
		fmt.Println("[RecupDatum] No Datum, Stop here")
		return nil
	}

	return value
}

func explore(conn net.PacketConn, p *Peer) {
	reqDatum := make([]RequestDatum, 0)
	reqDatum = append(reqDatum, buildRequestDatum(p.Name, p.Root.Hash, 0))

	for len(reqDatum) != 0 {

		//Pop the last element
		req := reqDatum[len(reqDatum)-1]
		reqDatum = reqDatum[:len(reqDatum)-1]

		//Recup the Datum
		value := RecupDatum(conn, &req, p)
		//Error
		if value == nil {
			return
		}

		typeFile := value[0]
		value = value[1:]

		// Add the data in the tree
		fmt.Println("[Explore] Data from " + req.Path + " received !")
		pa := strings.Split(req.Path, "/")
		if p.Root.Hash != req.Hash {
			change := AddNode(p.Root, pa[1:], pa[len(pa)-1], req.Hash, typeFile)
			if !change {
				continue
			}
		}

		// If it's directory, need to explore its children
		if typeFile == DIRECTORY {
			fmt.Println("[Explore] Directory received !")

			for i := len(value) - 64; i >= 0; i -= 64 {
				name := string(bytes.TrimRight(value[i:i+32], string(byte(0))))
				hash := value[i+32 : i+64]
				path := req.Path + "/" + name
				fmt.Println(path)

				reqDatum = append(reqDatum, RequestDatum{Path: path, Hash: [32]byte(hash), Count: 0})
			}

		} else {
			fmt.Println("[Explore] File received")
		}

	}
}

func download(conn net.PacketConn, p *Peer, first_hash [32]byte, start_path string) {
	reqDatum := make([]RequestDatum, 0)
	reqDatum = append(reqDatum, buildRequestDatum(start_path, first_hash, 0))

	for len(reqDatum) != 0 {
		//Pop the last element
		req := reqDatum[len(reqDatum)-1]
		reqDatum = reqDatum[:len(reqDatum)-1]

		//Recup the Datum
		value := RecupDatum(conn, &req, p)
		//Error
		if value == nil {
			return
		}

		typeFile := value[0]
		value = value[1:]

		if typeFile == DIRECTORY {
			fmt.Println("[Download] Directory received !")
			err := os.MkdirAll(req.Path, os.ModePerm)
			if err != nil {
				fmt.Println("Error mkdir all :", err.Error())
			}
			for i := len(value) - 64; i >= 0; i -= 64 {
				name := string(bytes.TrimRight(value[i:i+32], string(byte(0))))
				hash := value[i+32 : i+64]
				path := req.Path + "/" + name
				fmt.Println(path)

				reqDatum = append(reqDatum, RequestDatum{Path: path, Hash: [32]byte(hash), Count: 0})
			}

		} else if typeFile == TREE {
			fmt.Println("[Download] BigFile received !")

			for i := len(value) - 32; i >= 0; i -= 32 {
				hash := value[i : i+32]
				reqDatum = append(reqDatum, RequestDatum{Path: req.Path, Hash: [32]byte(hash), Count: req.Count})
			}

		} else if typeFile == CHUNK {

			file, err := os.OpenFile(req.Path, os.O_WRONLY|os.O_CREATE, os.ModePerm)
			if err != nil {
				fmt.Println("Error open file :", err.Error())
				return
			}
			_, err = file.WriteAt(value, req.Count)
			if err != nil {
				fmt.Println("Error writeAt :", err.Error())
				return
			}

			if len(reqDatum) > 0 && reqDatum[len(reqDatum)-1].Path == req.Path {
				reqDatum[len(reqDatum)-1].Count = req.Count + int64(len(value))
			}

		} else {
			fmt.Println("Error : unknown file type !")
			return
		}
	}
}
