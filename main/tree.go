package main

import "fmt"

type Node struct {
	FileType byte
	Hash     [32]byte
	Name     string // ->
	Children []*Node
}

func add_node(n *Node, path []string, name string, hash [32]byte, type_file byte) *Node {
	fmt.Println(path)
	if n == nil {
		return &Node{FileType: type_file, Hash: hash, Name: name, Children: []*Node(make([]*Node, 0))}
	}

	if len(path) == 1 {
		n.Children = append(n.Children, add_node(nil, path[1:], name, hash, type_file))
		return n
	}

	for i := 0; i < len(n.Children); i++ {
		if n.Children[i].Name == path[0] {
			n.Children[i] = add_node(n.Children[i], path[1:], name, hash, type_file)
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
				print_node(n.Children[i])
			}
		}
	}

}
