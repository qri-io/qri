package dsgraph

var walkParallelism = 4

type NodeType string

var (
	NtDataset       = NodeType("dataset")
	NtMetadata      = NodeType("metadata")
	NtCommit        = NodeType("commit")
	NtData          = NodeType("data")
	NtQuery         = NodeType("query")
	NtAbstQuery     = NodeType("abst_query")
	NtStructure     = NodeType("structure")
	NtAbstStructure = NodeType("abst_structure")
	NtNamespace     = NodeType("namespace")
)

// Node is a typed reference to a path
type Node struct {
	Type  NodeType
	Path  string
	Links []Link
}

func (a Node) Equal(b *Node) bool {
	return a.Type == b.Type && a.Path == b.Path
}

func (n *Node) AddLinks(links ...Link) {
ADDITIONS:
	for _, link := range links {
		for _, l := range n.Links {
			if link.To.Path == "" || link.Equal(l) {
				continue ADDITIONS
			}
		}
		n.Links = append(n.Links, link)
	}
}

// TODO - still considering if links need to be typed or not
// type LinkType string

// var (
// 	LtPrevious      = LinkType("previous")
// 	LtResource      = LinkType("resource")
// 	LtDsData        = LinkType("dataset_data")
// 	LtDsCommit      = LinkType("dataset_commit")
// 	LtAbstStructure = LinkType("abst_structure")
// 	LtQuery         = LinkType("query")
// 	LtAbstQuery     = LinkType("abst_query")
// 	LtNamespaceTip  = LinkType("namespace_tip")
// )

// Link is a typed, directional connection from one
// node to another
type Link struct {
	// Type     LinkType
	From, To *Node
}

func (a Link) Equal(b Link) bool {
	return a.From.Equal(b.From) && a.To.Equal(b.To)
}

func FilterNodeTypes(graph *Node, nodetypes ...NodeType) (nodes []*Node) {
	Walk(graph, 0, func(n *Node) error {
		if n != nil {
			for _, nt := range nodetypes {
				if n.Type == nt {
					nodes = append(nodes, n)
					break
				}
			}
		}
		return nil
	})
	return
}

func Walk(node *Node, depth int, visit func(n *Node) error) error {
	if err := visit(node); err != nil {
		return err
	}
	for _, l := range node.Links {
		if err := Walk(l.To, depth+1, visit); err != nil {
			return err
		}
	}
	return nil
}
