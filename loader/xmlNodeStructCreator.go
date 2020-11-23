package loader

import (
	"bufio"
	"os"
)

type label struct {
	parentNode   *Node
	hasChildNode bool
	isRepeat     bool
}

// Levelmap 层级map
var Levelmap = make(map[int]map[string]*label)

// CreateLevelMap 创建层级map
func CreateLevelMap(node *Node) {

	if node == nil {

		return
	}

	var hasCnodes bool
	cnodes := node.childNodes

	if cnodes != nil && len(cnodes) > 0 {

		hasCnodes = true

	} else {

		hasCnodes = false
	}

	labelmap, ok := Levelmap[node.level]

	if !ok {

		labelmap = make(map[string]*label)
	}

	if l, ok := labelmap[node.name]; !ok {

		labelmap[node.name] = &label{
			parentNode:   node.parentNode,
			hasChildNode: hasCnodes,
			isRepeat:     false,
		}

	} else {

		if hasCnodes {

			l.hasChildNode = hasCnodes

			if l.parentNode == node.parentNode {

				l.isRepeat = true

			} else {

				l.parentNode = node.parentNode
			}
		}
	}

	Levelmap[node.level] = labelmap

	for _, nd := range cnodes {

		CreateLevelMap(nd)
	}

}

// CreateNodeStruct 创建结构体
func CreateNodeStruct() {

	file, err := os.OpenFile("xmlnodes.go", os.O_WRONLY|os.O_CREATE, 0666)
	defer file.Close()

	if err != nil {

		return
	}

	wr := bufio.NewWriter(file)
	wr.WriteString("package xmlnodes\n")

	levelcount := len(Levelmap)

	if levelcount > 0 {

		for i := 0; i < levelcount; i++ {

			labelmap := Levelmap[i]

			if i+1 < levelcount {

				for key := range labelmap {

					l := labelmap[key]

					if !l.hasChildNode {
						continue
					}

					wr.WriteString("type " + key + " struct {\n")
					wr.WriteString("    XMLName xml.Name `xml:\"" + key + "\"`\n")
					nextLevelLabelMap := Levelmap[i+1]

					for hlkey := range nextLevelLabelMap {

						temp := nextLevelLabelMap[hlkey]

						if temp.parentNode.name == key {

							if temp.isRepeat {

								wr.WriteString("    " + hlkey + "List []" + hlkey + " `xml:\"" + hlkey + "\"`\n")

							} else {

								if temp.hasChildNode {

									wr.WriteString("    " + hlkey + " " + hlkey + " `xml:\"" + hlkey + "\"`\n")

								} else {

									wr.WriteString("    " + hlkey + " string `xml:\"" + hlkey + "\"`\n")
								}

							}
						}
					}

					wr.WriteString("}\n")
				}
			}
		}
	}

	wr.Flush()
}
