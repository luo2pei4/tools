package loader

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

// stack 深度为10的栈，用来压入XML的标签名称。
type stack struct {
	MaxTop int      // 栈顶最大值
	Top    int      // 栈顶标识
	arr    []string // 数组
}

func (s *stack) push(name string) error {

	if s.Top == s.MaxTop-1 {
		return errors.New("push(), stack full")
	}

	s.Top++
	s.arr[s.Top] = name

	return nil
}

func (s *stack) pop() (name string, err error) {

	if s.Top == -1 {
		return "", errors.New("pop(), stack empty")
	}

	name = s.arr[s.Top]
	s.Top--

	return name, nil
}

func initStack(MaxTop, Top int) *stack {

	// 创建栈对象，最多支持16层嵌套
	return &stack{
		MaxTop: 16,
		Top:    -1,
		arr:    make([]string, 16),
	}
}

func (s *stack) topValue() (name string, err error) {

	if s.Top == -1 {
		return "", errors.New("topValue(),stack empty")
	}

	name = s.arr[s.Top]

	return name, nil
}

// previousValue 获取数组倒数第二个值
func (s *stack) previousValue() (name string, level int, err error) {

	if s.Top-1 == -1 {
		return "", 0, nil
	}

	return s.arr[s.Top-1], s.Top - 1, nil
}

// Node XML的节点结构体
type Node struct {
	name       string            // 标签名称
	attr       map[string]string // 标签属性Map
	value      string            // 节点的值
	level      int               // 节点所在层级
	parentNode *Node             // 父节点
	childNodes []*Node           // 子节点切片
}

// PrintNode 在控制台输出节点左右信息
func (node *Node) PrintNode() {

	var attrbutes string = ""

	if len(node.attr) > 0 {

		attrbutes = "ATTR: "

		for key, value := range node.attr {

			attrbutes = attrbutes + key + "-" + value + "; "
		}
	}

	fmt.Println(node.name, node.value, attrbutes)

	childNodes := node.childNodes

	if len(childNodes) != 0 {

		for _, childNode := range childNodes {

			childNode.PrintNode()
		}
	}
}

// 获取当前节点的父节点
func getParentNode(stack *stack, node *Node, previousName string, previousLevel int) *Node {

	if node.name != previousName {

		for _, childNode := range node.childNodes {

			node = getParentNode(stack, childNode, previousName, previousLevel)
		}
	}

	return node
}

// LoadXML 通过文件路径加载xml，并返回解析的节点链
func LoadXML(xmlDec *xml.Decoder) *Node {

	// 初始化栈
	s := initStack(10, -1)

	// 创建节点链
	root := new(Node)

	// 声明父节点和当前节点
	// 声明父节点用于保存父节点，减少递归调用寻找父节点的次数
	var parentNode *Node
	var currentNode *Node

	var startElementLoaded bool = false

	for {

		// 获取xml的token。
		// ** 这个地方比较坑，标签和标签之间如果有换行，换行符和下一行开始标签开始之前的空格会识别为xml.CharData类型。
		token, err := xmlDec.Token()

		if err == io.EOF {
			err = nil
			break
		}

		if err != nil {
			panic(err)
		}

		// 判断节点类型
		switch t := token.(type) {

		// 开始标签，主要进行标签名压栈和添加父子节点链接
		case xml.StartElement:

			name := t.Name.Local
			s.push(name)
			var attr map[string]string

			if len(t.Attr) != 0 {

				attr = make(map[string]string)

				for _, xmlAttr := range t.Attr {
					attr[xmlAttr.Name.Local] = xmlAttr.Value
				}
			}

			if s.Top == 0 {

				root.name = name
				root.attr = attr
				root.level = s.Top
				root.childNodes = make([]*Node, 0)
				parentNode = root

			} else {

				currentNode = &Node{
					name:       name,
					attr:       attr,
					level:      s.Top,
					childNodes: make([]*Node, 0),
				}

				previousName, previousLevel, _ := s.previousValue()

				if parentNode.name != previousName {
					parentNode = getParentNode(s, parentNode, previousName, previousLevel)
				}

				currentNode.parentNode = parentNode
				parentNode.childNodes = append(parentNode.childNodes, currentNode)

				// 只有在非根节点的开始元素载入时才会设置为true
				startElementLoaded = true
			}

		// 将值存入节点链中最后一个节点
		case xml.CharData:

			// 根节点后面的换行符会被识别为xml.CharData类型，currentNode还没有被实例化，
			// 此时向currentNode写入值会产生panic，所以在此处要做一个非nil的判断
			if currentNode != nil {

				// 判断是否已经载入了开始元素
				if startElementLoaded {

					// 处理开始元素后面的换行符和第二行的空格
					value := string(t)
					value = strings.Replace(value, "\n", "", -1)
					currentNode.value = value
				}
			}

		// 结束标签，主要将当前标签名弹出栈
		case xml.EndElement:
			end := t.Name.Local
			top, err := s.topValue()

			if err != nil {
				fmt.Println("xml.EndElement ", err.Error())
			}

			if end == top {
				s.pop()
			}

			startElementLoaded = false

		// Comment，ProcInst和Directive类型的内容不做处理
		case xml.Comment:
		case xml.ProcInst:
		case xml.Directive:
		default:
			panic(errors.New("parse failed"))
		}
	}

	return root
}
