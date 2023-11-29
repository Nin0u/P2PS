package main

import "fmt"

type Node struct {
	FileType byte
	Hash     [32]byte
	Name     string // ->
	Children [32]*Node
}

func add_node(n *Node, path []string, name string, hash [32]byte, filetype byte) *Node {
	if n == nil {
		return &Node{FileType: filetype, Hash: hash, Name: name, Children: [32]*Node(make([]*Node, 32))}
	}

	if len(path) == 1 {
		n.Children[len(n.Children)-1] = add_node(n.Children[len(n.Children)-1], path[1:], name, hash, filetype)
		return n
	}

	for i := 0; i < len(n.Children); i++ {
		if n.Children[i].Name == path[0] {
			n.Children[i] = add_node(n.Children[i], path[1:], name, hash, filetype)
			break
		}
	}
	return n
}

func print_node(n *Node) {
	fmt.Println(n.Name, ": ")
	if n.FileType == DIRECTORY {
		for i := 0; i < len(n.Children); i++ {
			fmt.Println(" - ", n.Children[i].Name)
		}

		for i := 0; i < len(n.Children); i++ {
			if n.Children[i].FileType == DIRECTORY {
				print_node(n)
			}
		}
	}

}
