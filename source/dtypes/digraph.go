package dtypes

import (
	"fmt"
	"github.com/wk8/go-ordered-map/v2"
)

type Digraph = orderedmap.OrderedMap[string, Set[string]]

func String(D *Digraph) string {
	result := "{\n"
	for pair := D.Oldest(); pair != nil; pair = pair.Next() {
		result += fmt.Sprintf("%v : %v", pair.Key, pair.Value.String())
	}
	result += "}\n"
	return result
}

// Used for performing the Tarjan sort.
type data struct {
	graph  *Digraph
	nodes  []node
	stack  []string
	index  map[string]int
	output [][]string
}

type node struct {
	lowlink int
	stacked bool
}

// This partitions the graph into strongly-connected components, while performing a
// topological search on them.
func Tarjan(graph *Digraph)  [][]string {
	data := &data{
		graph: graph,
		nodes: make([]node, 0, graph.Len()),
		index: make(map[string]int, graph.Len()),
	}
	for pair := graph.Oldest(); pair != nil; pair = pair.Next() {
		if _, ok := data.index[pair.Key]; !ok {
			data.getStronglyConnectedComponent(pair.Key)
		}
	}
	return data.output
}

func (data *data) getStronglyConnectedComponent(v string) *node {
	index := len(data.nodes)
	data.index[v] = index
	data.stack = append(data.stack, v)
	data.nodes = append(data.nodes, node{lowlink: index, stacked: true})
	node := &data.nodes[index]
	R, _ := data.graph.Get(v)
	for w := range R {
		i, seen := data.index[w]
		if !seen {
			n := data.getStronglyConnectedComponent(w)
			if n.lowlink < node.lowlink {
				node.lowlink = n.lowlink
			}
		} else if data.nodes[i].stacked {
			if i < node.lowlink {
				node.lowlink = i
			}
		}
	}

	if node.lowlink == index {
		var vertices []string
		i := len(data.stack) - 1
		for {
			w := data.stack[i]
			stackIndex := data.index[w]
			data.nodes[stackIndex].stacked = false
			vertices = append(vertices, w)
			if stackIndex == index {
				break
			}
			i--
		}
		data.stack = data.stack[:i]
		data.output = append(data.output, vertices)
	}

	return node
}

func SetOfNodes(D *Digraph) *Set[string] {
	result := Set[string]{}
	for pair := D.Oldest(); pair != nil; pair = pair.Next() {
		result.Add(pair.Key)
	}
	return &result
}

func GetArbitraryNode(D *Digraph) (string, bool) {
	return D.Oldest().Key, D.Len() != 0
}

// This adds an arrow with transitive closure to a digraph, on the assumption that it is
// already transitively closed.
func AddTransitiveArrow(D *Digraph, a, b string) {
	if !SetOfNodes(D).Contains(b) {
		D.Set(b, Set[string]{})
	}
	if !SetOfNodes(D).Contains(a) {
		D.Set(a, Set[string]{})
	}
	AddArrow(D, a, b)
	
	// Note again that we depend on the digraph already being transitiviely
	// closed.
	// So we don't have to look recursively through the graph to find what
	// b transitively leads to, because those are already its immmediate
	// neighbors.
	arrowsFromB, _ := D.Get(b)
	for k := range arrowsFromB {
		AddArrow(D, a, k)
	}
	for k := range ArrowsTo(D, a) {
		AddArrow(D, k, b)
		ns, _ := D.Get(b)
		for v := range ns {
			AddArrow(D, k, v)
		}
	}
}

// This supposes that the nodes already exist.
func AddArrow(D *Digraph, a, b string) {
	neighbors, _ := D.Get(a)
	neighbors.Add(b)
	D.Set(a, neighbors)
}

func  ArrowsTo(D *Digraph, x string) Set[string] {
	target := Set[string]{}
	target.Add(x)
	results := Set[string]{}
	for {
		newResults := false
		for pair := D.Oldest(); pair != nil; pair = pair.Next() {
			if pair.Value.OverlapsWith(target) {
				if !results.Contains(pair.Key) {
					results.Add(pair.Key)
					newResults = true
				}
			}
		}
		if !newResults {
			break
		}
	}
	return results
}

func Add(D *Digraph, node string, neighbors []string) {
	s := MakeFromSlice(neighbors)
	D.Set(node, s)
}



func Index[E comparable](slice []E, element E) int {
	result := -1
	for k, v := range slice {
		if v == element {
			result = k
			break
		}
	}
	return result
}
