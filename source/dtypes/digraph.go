package dtypes

import (
	"fmt"

	"github.com/tim-hardcastle/pipefish/source/dtypes"
	"github.com/wk8/go-ordered-map/v2"
)

type Digraph = orderedmap.OrderedMap[string, Set[string]]

func String(D *Digraph) string {
	result := "{\n"
	for pair := D.Oldest(); pair != nil; pair = pair.Next() {
		result += fmt.Sprintf("%v : %v", pair, v.String())
	}
	result += "}\n"
	return result
}

// Used for performing the Tarjan sort.
type data struct {
	graph  Digraph
	nodes  []node
	stack  []string
	index  map[string]int
	output [][]string
}

type node struct {
	lowlink int
	stacked bool
}

// This partitions the graph into strongly-connected components
func Tarjan(graph Digraph)  [][]string {
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

func (data *data) getStronglyConnectedComponent(v E) *node {
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

func (D Digraph) GetArbitraryNode() (string, bool) {
	return D.Oldest().Key, D.Len() != 0
}

// This checks to see if a node already has an entry before adding it to the digraph.
func AddSafe(D *Digraph, node string, neighbors []string) bool {
	if !SetOfNodes(D).Contains(node) {
		neighborSet := Set[string]{}
		for _, neighbor := range neighbors {
			neighborSet = neighborSet.Add(neighbor)
		}
		D.Set(node, neighborSet)
		return true
	}
	return false
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
	(D)[a].Add(b)
	(D)[a].AddSet((D)[b])
	for e := range *(D.ArrowsTo(a)) {
		(D)[string].Add(b)
		(D)[string].AddSet((D)[b])
	}
}

// This supposes that the nodes already exist.
func AddArrow(D *Digraph, a, b string) {

}

func  ArrowsTo(D Digraph, e string) *Set[string] {
	result := Set[string]{}
	for k, V := range D {
		if V.Contains(e) {
			result.Add(k)
		}
	}
	return &result
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

func (T *Digraph[string]) PointsTo(candidate, target E) bool {
	return (*T)[candidate].Contains(target)
}
