package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

// Keys are Hash, Value are the value in the getDatum request

type Data struct {
	Value        []byte
	LastTimeUsed time.Time
}

type DatumCache struct {
	content map[[32]byte]Data
	mutex   sync.Mutex
}

var datumCache DatumCache = DatumCache{content: make(map[[32]byte]Data)}
var debug_download = false

// TODO : On part pour l'instant du principe que la taille  est limitée par le temps mais à voir avec le multi DL
func AddDatumCache(hash [32]byte, value []byte) {
	DatumCacheClearer()
	datumCache.mutex.Lock()
	datumCache.content[hash] = Data{Value: value, LastTimeUsed: time.Now()}
	datumCache.mutex.Unlock()
}

func GetDatumCache(hash [32]byte) ([]byte, bool) {
	datumCache.mutex.Lock()
	value, ok := datumCache.content[hash]
	value.LastTimeUsed = time.Now()
	datumCache.mutex.Unlock()
	return value.Value, ok
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
	Info  int16 // 0 -> ok, -1 -> Error, 1 -> Chunk dl need to write
}

func buildRequestDatum(path string, hash [32]byte, count int64) RequestDatum {
	return RequestDatum{Path: path, Hash: hash, Count: count, Info: 0}
}

func RecupDatum(conn net.PacketConn, req *RequestDatum, p *Peer) []byte {
	//Recup the Datum
	value, ok := GetDatumCache(req.Hash)

	if !ok {
		// TODO : on choisit quelle addr du pair ??? J'ai mis [0] pour l'instant
		_, err := sendGetDatum(conn, p.Addr[0], req.Hash)

		if err != nil {
			color.Red("[RecupDatum] Error send getDatum : %s\n", err.Error())
			return nil
		}

		value, ok = GetDatumCache(req.Hash)
	}

	if !ok {
		color.Red("[RecupDatum] Error on get datum %s\n", req.Path)
		return nil
	}

	if value == nil {
		color.Magenta("[RecupDatum] No Datum, Stop here\n")
		return nil
	}

	return value
}

func Explore(conn net.PacketConn, p *Peer) error {
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
			return errors.New("error RecupDatum")
		}

		typeFile := value[0]
		value = value[1:]

		// Add the data in the tree
		if debug_download {
			fmt.Println("[Explore] Data from " + req.Path + " received !")
		}
		pa := strings.Split(req.Path, "/")
		if p.Root.Hash != req.Hash {
			change := AddNode(p.Root, pa[1:], pa[len(pa)-1], req.Hash, typeFile)
			if !change {
				continue
			}
		}

		// If it's directory, need to explore its children
		if typeFile == DIRECTORY {
			if debug_download {
				fmt.Println("[Explore] Directory received !")
			}

			for i := len(value) - 64; i >= 0; i -= 64 {
				name := string(bytes.TrimRight(value[i:i+32], string(byte(0))))
				hash := value[i+32 : i+64]
				path := req.Path + "/" + name

				reqDatum = append(reqDatum, buildRequestDatum(path, [32]byte(hash), 0))
			}

		} else {
			if debug_download {
				fmt.Println("[Explore] File received")
			}
		}
	}
	return nil
}

func Download(conn net.PacketConn, p *Peer, first_hash [32]byte, start_path string) {
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
			if debug_download {
				fmt.Println("[Download] Directory received !")
			}

			err := os.MkdirAll(req.Path, os.ModePerm)
			if err != nil {
				color.Red("[Download] Error mkdir all : %s\n", err.Error())
			}
			for i := len(value) - 64; i >= 0; i -= 64 {
				name := string(bytes.TrimRight(value[i:i+32], string(byte(0))))
				hash := value[i+32 : i+64]
				path := req.Path + "/" + name
				fmt.Println(path)

				reqDatum = append(reqDatum, buildRequestDatum(path, [32]byte(hash), 0))
			}

		} else if typeFile == TREE {
			if debug_download {
				fmt.Println("[Download] BigFile received !")
			}

			for i := len(value) - 32; i >= 0; i -= 32 {
				hash := value[i : i+32]
				reqDatum = append(reqDatum, buildRequestDatum(req.Path, [32]byte(hash), req.Count))
			}

		} else if typeFile == CHUNK {
			if debug_download {
				fmt.Println("[Download] Chunk received !")
			}

			file, err := os.OpenFile(req.Path, os.O_WRONLY|os.O_CREATE, os.ModePerm)
			if err != nil {
				color.Red("[Download] Error open file : %s\n", err.Error())
				return
			}
			_, err = file.WriteAt(value, req.Count)
			if err != nil {
				color.Red("[Download] Error writeAt : %s\n", err.Error())
				return
			}

			if len(reqDatum) > 0 && reqDatum[len(reqDatum)-1].Path == req.Path {
				reqDatum[len(reqDatum)-1].Count = req.Count + int64(len(value))
			}

		} else {
			color.Magenta("[Download] Error : unknown file type !\n")
			return
		}
	}
}

func download_multi_aux(conn net.PacketConn, req *RequestDatum, p *Peer) []RequestDatum {
	buff := make([]RequestDatum, 0)

	//Recup the Datum
	value := RecupDatum(conn, req, p)
	//Error
	if value == nil {
		buff = append(buff, RequestDatum{Info: -1})
		return buff
	}

	typeFile := value[0]
	value = value[1:]

	if typeFile == DIRECTORY {
		if debug_download {
			fmt.Println("[DownloadMultiAux] Directory received !")
		}

		err := os.MkdirAll(req.Path, os.ModePerm)
		if err != nil {
			color.Red("[DownloadMultiAux] Error mkdir all : %s\n", err.Error())
		}
		for i := len(value) - 64; i >= 0; i -= 64 {
			name := string(bytes.TrimRight(value[i:i+32], string(byte(0))))
			hash := value[i+32 : i+64]
			path := req.Path + "/" + name

			buff = append(buff, RequestDatum{Path: path, Hash: [32]byte(hash), Count: 0, Info: 0})
		}

	} else if typeFile == TREE {
		if debug_download {
			fmt.Println("[DownloadMultiAux] BigFile received !")
		}

		for i := len(value) - 32; i >= 0; i -= 32 {
			hash := value[i : i+32]
			buff = append(buff, RequestDatum{Path: req.Path, Hash: [32]byte(hash), Count: req.Count, Info: 0})
		}

	} else if typeFile == CHUNK {
		if debug_download {
			fmt.Println("[DownloadMultiAux] Chunk received !")
		}
		req.Info = 1
		buff = append(buff, *req)
	} else {
		color.Magenta("[DownloadMultiAux] Error : unknown file type !\n")
		buff = append(buff, RequestDatum{Info: -1})
		return buff
	}
	return buff
}

func download_multi(conn net.PacketConn, p *Peer, first_hash [32]byte, start_path string) error {
	const max_request = 32

	reqDatum := make([]RequestDatum, 0)
	reqDatum = append(reqDatum, buildRequestDatum(start_path, first_hash, 0))

	for len(reqDatum) != 0 {

		//Pop the max_request last elements
		nb := min(max_request, len(reqDatum))
		reqs := reqDatum[len(reqDatum)-nb:]
		reqDatum = reqDatum[:len(reqDatum)-nb]

		//Demand the next nodes by multithreading
		var buffs = [max_request][]RequestDatum{}
		var wg sync.WaitGroup
		wg.Add(nb)
		for i := 0; i < nb; i++ {

			req := &reqs[i]

			go func(i int) {
				defer wg.Done()
				buffs[i] = download_multi_aux(conn, req, p)
			}(i)
		}
		wg.Wait()

		//Add the result to the stack
		for i := 0; i < nb; i++ {
			if len(buffs[i]) > 0 && buffs[i][0].Info == -1 {
				color.Red("[DownloadMulti] Error no value\n")
				return errors.New("error no value")
			}
			reqDatum = append(reqDatum, buffs[i]...)
		}

		// Write chunks
		for {
			if len(reqDatum) == 0 {
				break
			}

			// If the chunk has been downloaded
			if reqDatum[len(reqDatum)-1].Info == 1 {
				//Pop and write the element
				req := reqDatum[len(reqDatum)-1]
				reqDatum = reqDatum[:len(reqDatum)-1]

				value := RecupDatum(conn, &req, p)
				value = value[1:]

				file, err := os.OpenFile(req.Path, os.O_WRONLY|os.O_CREATE, os.ModePerm)
				if err != nil {
					color.Red("[DownloadMulti] Error open file : %s\n", err.Error())
					return errors.New("error open")
				}
				_, err = file.WriteAt(value, req.Count)
				if err != nil {
					color.Red("[DownloadMulti] Error writeAt : %s\n", err.Error())
					return errors.New("error writeAt")
				}

				if len(reqDatum) > 0 && reqDatum[len(reqDatum)-1].Path == req.Path {
					reqDatum[len(reqDatum)-1].Count = req.Count + int64(len(value))
				}

			} else {
				break
			}
		}
	}
	return nil
}
