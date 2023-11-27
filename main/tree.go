package main

type Node struct {
	type_file byte
	hash      [32]byte
	name      string // ->
	children  [32]*Node
}

func add_node(n *Node, path []string, name string, hash [32]byte, type_file byte) *Node {
	if n == nil {
		return &Node{type_file: type_file, hash: hash, name: name, children: [32]*Node(make([]*Node, 32))}
	}
	return nil
}
