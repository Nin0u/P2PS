package main

import (
	"errors"
	"fmt"
)

type Node struct {
	FileType byte
	Hash     [32]byte
	Name     string
	Children []*Node
}

func BuildNode(name string, hash [32]byte, type_file byte) *Node {
	return &Node{FileType: type_file, Hash: hash, Name: name, Children: []*Node(make([]*Node, 0))}
}

// Return false no change
// Return true if change
func AddNode(n *Node, path []string, name string, hash [32]byte, type_file byte) bool {
	if len(path) == 1 {
		for i := 0; i < len(n.Children); i++ {
			if n.Children[i].Name == name {
				if n.Children[i].Hash == hash {
					return false
				} else {
					n.Children[i] = BuildNode(name, hash, type_file)
					return true
				}
			}
		}
		n.Children = append(n.Children, BuildNode(name, hash, type_file))
		return true
	}

	for i := 0; i < len(n.Children); i++ {
		if n.Children[i].Name == path[0] {
			return AddNode(n.Children[i], path[1:], name, hash, type_file)
		}
	}
	return false
}

func PrintNode(n *Node, prefix string) {
	fmt.Println(prefix+n.Name, ": ")
	if n.FileType == DIRECTORY {
		for i := 0; i < len(n.Children); i++ {
			fmt.Println(" - ", n.Children[i].Name)
		}

		for i := 0; i < len(n.Children); i++ {
			if n.Children[i].FileType == DIRECTORY {
				PrintNode(n.Children[i], prefix+n.Name+"/")
			}
		}
	}
}

// Gets the hash of a file in the tree
func FindPath(n *Node, path []string) ([32]byte, error) {
	if len(path) == 0 {
		return n.Hash, nil
	}

	for i := 0; i < len(n.Children); i++ {
		if n.Children[i].Name == path[0] {
			return FindPath(n.Children[i], path[1:])
		}
	}

	return n.Hash, errors.New("not Found")
}
