package dag

import (
	// "errors"
	"fmt"
	"sync"

	"github.com/alchematik/athanor/internal/set"
)

func NewGraph() *Graph {
	return &Graph{
		forwardEdges:  map[string]*set.Set[string]{},
		backwardEdges: map[string]*set.Set[string]{},
	}
}

type Graph struct {
	forwardEdges  map[string]*set.Set[string]
	backwardEdges map[string]*set.Set[string]
}

func (g *Graph) AddEdge(from, to string) error {
	// if from == "" || to == "" {
	// 	return errors.New("node names must not be empty")
	// }

	forward, ok := g.forwardEdges[from]
	if !ok {
		forward = set.NewSet[string]()
		g.forwardEdges[from] = forward
	}
	forward.Add(to)

	backward, ok := g.backwardEdges[to]
	if !ok {
		backward = set.NewSet[string]()
		g.backwardEdges[to] = backward
	}
	backward.Add(from)

	// Add empty value to make sure from has a backward edge.
	if _, ok := g.backwardEdges[from]; !ok {
		g.backwardEdges[from] = set.NewSet[string]()
	}
	if _, ok := g.forwardEdges[to]; !ok {
		g.forwardEdges[to] = set.NewSet[string]()
	}

	return nil
}

type Iterator struct {
	sync.Mutex

	graph   *Graph
	next    *set.Set[string]
	visited map[string]bool
	deps    map[string]*set.Set[string]
}

func InitIterator(g *Graph) *Iterator {
	next := set.NewSet[string]()
	deps := map[string]*set.Set[string]{}
	for n, edges := range g.backwardEdges {
		deps[n] = edges.Clone()

		if edges.Len() == 0 {
			next.Add(n)
		}
	}

	iter := &Iterator{
		graph:   g,
		next:    next,
		visited: map[string]bool{},
		deps:    deps,
	}
	return iter
}

func (iter *Iterator) Next() []string {
	iter.Lock()
	defer iter.Unlock()

	var nextNodes []string
	for _, n := range iter.next.Values() {
		nextNodes = append(nextNodes, n)
	}

	iter.next = set.NewSet[string]()

	return nextNodes
}

func (iter *Iterator) Start(node string) error {
	iter.visitNode(node, true)

	forward := iter.forwardEdges(node)

	if forward.Len() == 0 {
		iter.addNext(node)
		return nil
	}

	for _, e := range forward.Values() {
		// Reset forward edges to not visited so they get processed again.
		if iter.Visited(e) {
			iter.visitNode(e, false)
		}

		deps := iter.dependencies(e)
		deps.Remove(node)

		if deps.Len() == 0 {
			iter.addNext(e)
		}

		// Add dependency so that when all child nodes are processed, we process the parent node again.
		nodeDeps := iter.dependencies(node)
		nodeDeps.Add(e)
	}

	return nil
}

func (iter *Iterator) Done(node string) error {
	if !iter.Visited(node) {
		return fmt.Errorf("done called on %q when not visited", node)
	}

	back := iter.backwardEdges(node)

	for _, e := range back.Values() {
		deps := iter.dependencies(e)

		deps.Remove(node)
		if deps.Len() == 0 {
			iter.addNext(e)
		}
	}

	return nil
}

func (iter *Iterator) Visited(node string) bool {
	iter.Lock()
	defer iter.Unlock()

	return iter.visited[node]
}

func (iter *Iterator) addNext(node string) {
	iter.Lock()
	defer iter.Unlock()

	iter.next.Add(node)
}

func (iter *Iterator) forwardEdges(node string) *set.Set[string] {
	iter.Lock()
	defer iter.Unlock()

	return iter.graph.forwardEdges[node]
}

func (iter *Iterator) backwardEdges(node string) *set.Set[string] {
	iter.Lock()
	defer iter.Unlock()

	return iter.graph.backwardEdges[node]
}

func (iter *Iterator) visitNode(node string, visited bool) {
	iter.Lock()
	defer iter.Unlock()

	iter.visited[node] = visited
}

func (iter *Iterator) dependencies(node string) *set.Set[string] {
	iter.Lock()
	defer iter.Unlock()

	deps, ok := iter.deps[node]
	if !ok {
		deps = set.NewSet[string]()
		iter.deps[node] = deps
	}

	return deps
}
