package main

type ExportNode struct {
	Path     string
	Hash     [32]byte
	Num      int64
	Children []*ExportNode
}

// Map containing Tree's Node. It serves to access efficately to the data ! Needed for handleGetDatum
var map_export map[[32]byte]*ExportNode = map[[32]byte]*ExportNode{}

func buildExportNode(path string, hash [32]byte, num int64) *ExportNode {
	node := ExportNode{Path: path, Hash: hash, Num: num}
	map_export[hash] = &node
	return &node
}

func exportFile(path string) {
	// file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	// if err != nil {
	// 	fmt.Println("[exportFile] error open", path, err.Error())
	// 	return
	// }

	// num := 0
	// buff := make([]*ExportNode, 0)
	// chunk := make([]byte, 1024)

	// //Cut the files in blocks
	// for {
	// 	n, err := file.Read(chunk)
	// 	if err != nil {
	// 		fmt.Println("[exportFile] error read", n, err.Error())
	// 		return
	// 	}

	// 	hash := sha256.Sum256(chunk)

	// 	buff = append(buff, buildExportNode(path, hash, int64(n)))
	// 	num += n

	// 	if n < 1024 {
	// 		break
	// 	}
	// }

	// //Build the tree
	// buff_bis := make([]*ExportNode, 0)

	// for {
	// 	children := make([]*ExportNode, 0)
	// 	for i := 0; i < 32 && i < len(buff); i++ {
	// 		children = append(children, buff[i])

	// 	}
	// }

}
