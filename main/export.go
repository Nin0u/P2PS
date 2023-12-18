package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"sync"
)

type ExportNode struct {
	Path     string
	Hash     [32]byte
	Num      int64
	Children []*ExportNode
	Type     byte
}

var rootExport *ExportNode = nil

// Map containing Tree's Node. It serves to access efficatively to the data ! Needed for handleGetDatum

type MapExport struct {
	Content map[[32]byte]*ExportNode
	Mutex   sync.Mutex
}

var map_export MapExport = MapExport{Content: map[[32]byte]*ExportNode{}}

func buildExportNode(path string, hash [32]byte, num int64, type_file byte) *ExportNode {
	node := ExportNode{Path: path, Hash: hash, Num: num, Type: type_file}
	map_export.Mutex.Lock()
	map_export.Content[hash] = &node
	map_export.Mutex.Unlock()
	return &node
}

func exportFile(path string) *ExportNode {
	fmt.Println("[exportFile]", path)
	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		fmt.Println("[exportFile] error open", path, err.Error())
		return nil
	}

	num := 0
	buff := make([]*ExportNode, 0)
	chunk := make([]byte, 1024)

	//Cut the files in blocks
	for {
		n, err := file.Read(chunk)
		if err != nil && err != io.EOF {
			fmt.Println("[exportFile] error read", n, err.Error())
			return nil
		}

		data := append([]byte{CHUNK}, chunk[:n]...)

		hash := sha256.Sum256(data)

		buff = append(buff, buildExportNode(path, hash, int64(num), CHUNK))
		num += n

		if n < 1024 {
			break
		}
	}

	//Build the tree
	buff_bis := make([]*ExportNode, 0)
	index := 0
	for {
		if len(buff) == 1 {
			break
		}

		children := make([]*ExportNode, 0)
		len_tab := min(32+index, len(buff))
		hashhash := make([]byte, 0)
		hashhash = append(hashhash, TREE)
		for i := 0; index < len_tab; i++ {
			children = append(children, buff[index])
			hashhash = append(hashhash, children[i].Hash[:]...)
			index++
		}

		hash_node := sha256.Sum256(hashhash)
		node := buildExportNode(path, hash_node, 0, TREE)
		buff_bis = append(buff_bis, node)

		if index == len(buff) {
			buff = buff_bis
			buff_bis = make([]*ExportNode, 0)
			index = 0
		}

		node.Children = children
	}

	return buff[0]

}

func exportDirectory(path string) *ExportNode {
	fmt.Println("[exportDirectory]", path)
	entry, err := os.ReadDir(path)
	if err != nil {
		fmt.Println("[exportDirectory] Error ReadDir ", path, err.Error())
		return nil
	}

	children := make([]*ExportNode, 0)

	hashhash := make([]byte, 0)
	hashhash = append(hashhash, DIRECTORY)
	for _, e := range entry {
		var node *ExportNode
		if e.IsDir() {
			node = exportDirectory(path + "/" + e.Name())
		} else {
			node = exportFile(path + "/" + e.Name())
		}

		children = append(children, node)
		name := [32]byte{}
		copy(name[:], []byte(e.Name()))
		hashhash = append(hashhash, name[:]...)
		hashhash = append(hashhash, node.Hash[:]...)
	}
	fmt.Println(hashhash)
	hash := sha256.Sum256(hashhash)
	node := buildExportNode(path, hash, 0, DIRECTORY)
	node.Children = children
	return node
}

func export(path string) {

	info, err := os.Stat(path)
	if err != nil {
		fmt.Println("[Export] Error stat", path, err.Error())
		return
	}

	//TODO: Vider la map !
	//TODO: Add mutex
	if info.IsDir() {
		rootExport = exportDirectory(path)
	} else {
		rootExport = exportFile(path)
	}
}

func writeExportTree(root *ExportNode) {
	if root.Type == CHUNK {
		fmt.Println("WRITE CHUNK", root.Num)
		f1, err := os.OpenFile(root.Path, os.O_RDONLY, os.ModePerm)
		if err != nil {
			fmt.Println("BLABLA :", err.Error())
			return
		}

		data := make([]byte, 1024)
		n, err := f1.ReadAt(data, root.Num)
		if err != nil && err != io.EOF {
			fmt.Println("[writeExportFile] error read", n, err.Error())
			return
		}

		f2, err := os.OpenFile("Test/"+root.Path, os.O_WRONLY|os.O_CREATE, os.ModePerm)
		if err != nil {
			fmt.Println("NANANA :", err.Error())
			return
		}

		_, err = f2.WriteAt(data[:n], root.Num)
		if err != nil {
			fmt.Println("GHGHGHG :", err.Error())
			return
		}

		f1.Close()
		f2.Close()
	} else {
		fmt.Println("Explore :", root.Hash)
		for i := 0; i < len(root.Children); i++ {
			writeExportTree(root.Children[i])
		}
	}
}
