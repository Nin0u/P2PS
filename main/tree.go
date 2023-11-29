package main

import "fmt"

type Node struct {
	type_file byte
	hash      [32]byte
	name      string // ->
	children  []*Node
}

func add_node(n *Node, path []string, name string, hash [32]byte, type_file byte) *Node {
	fmt.Println(path)
	if n == nil {
		return &Node{type_file: type_file, hash: hash, name: name, children: []*Node(make([]*Node, 0))}
	}

	if len(path) == 1 {
		n.children = append(n.children, add_node(nil, path[1:], name, hash, type_file))
		return n
	}

	for i := 0; i < len(n.children); i++ {
		if n.children[i].name == path[0] {
			n.children[i] = add_node(n.children[i], path[1:], name, hash, type_file)
			break
		}
	}
	return n
}

func print_node(n *Node) {
	fmt.Println(n.name, ": ")
	if n.type_file == DIRECTORY {
		for i := 0; i < len(n.children); i++ {
			fmt.Println(" - ", n.children[i].name)
		}

		for i := 0; i < len(n.children); i++ {
			if n.children[i].type_file == DIRECTORY {
				print_node(n.children[i])
			}
		}
	}

}
