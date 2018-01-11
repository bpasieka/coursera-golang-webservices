package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
)

const (
	dirNoSize = -1

	defaultNodePrefix      = "├───"
	lastNodePrefix         = "└───"
	defaultNodeLevelPrefix = "│\t"
	lastNodeLevelPrefix    = "\t"
	emptySizeSuffix        = " (empty)"
	nonEmptySizeSuffix     = " (%vb)"

	newLine = "\n"
)

type FileNode struct {
	name     string
	size     int
	children []FileNode
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"

	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	fileNodes, err := buildDirTree(path, []FileNode{}, printFiles)
	if err != nil {
		return err
	}
	drawDirTree(out, fileNodes, printFiles, "")

	return nil
}

func buildDirTree(path string, fileNodes []FileNode, printFiles bool) ([]FileNode, error) {
	// read directory entries (1 level)
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		// if a simple file and no need to process
		if !file.IsDir() && !printFiles {
			continue
		}

		if !file.IsDir() {
			fileNodes = append(fileNodes, FileNode{
				name: file.Name(),
				size: int(file.Size()),
			})
		} else {
			childrenFileNodes, err := buildDirTree(path+"/"+file.Name(), []FileNode{}, printFiles)
			if err != nil {
				return nil, err
			}

			fileNodes = append(fileNodes, FileNode{
				name:     file.Name(),
				size:     dirNoSize,
				children: childrenFileNodes,
			})
		}
	}

	return fileNodes, nil
}

func drawDirTree(out io.Writer, fileNodes []FileNode, printFiles bool, levelPrefix string) {
	for key, node := range fileNodes {
		length := len(fileNodes)
		nodePathName := levelPrefix

		if length > key+1 {
			nodePathName += defaultNodePrefix
		}

		if length == key+1 {
			nodePathName += lastNodePrefix
		}

		nodePathName += node.name
		if printFiles && node.size == 0 {
			nodePathName += emptySizeSuffix
		}

		if printFiles && node.size > 0 {
			nodePathName += fmt.Sprintf(nonEmptySizeSuffix, strconv.Itoa(node.size))
		}
		nodePathName += newLine
		out.Write([]byte(nodePathName))

		if len(node.children) > 0 {
			childrenPrefix := levelPrefix

			if length > key+1 {
				childrenPrefix += defaultNodeLevelPrefix
			}

			if length == key+1 {
				childrenPrefix += lastNodeLevelPrefix
			}

			drawDirTree(out, node.children, printFiles, childrenPrefix)
		}
	}
}
