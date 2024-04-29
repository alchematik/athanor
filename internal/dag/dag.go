package dag

import (
	"errors"
	"fmt"
	"sync"
)

func NewGraph() *Graph {
	return &Graph{
		forwardEdges:  map[string]*StringSet{},
		backwardEdges: map[string]*StringSet{},
	}
}

type Graph struct {
	forwardEdges  map[string]*StringSet
	backwardEdges map[string]*StringSet
}

func (g *Graph) AddEdge(from, to string) error {
	if from == "" || to == "" {
		return errors.New("node names must not be empty")
	}

	forward, ok := g.forwardEdges[from]
	if !ok {
		forward = NewStringSet()
		g.forwardEdges[from] = forward
	}
	forward.Add(to)

	backward, ok := g.backwardEdges[to]
	if !ok {
		backward = NewStringSet()
		g.backwardEdges[to] = backward
	}
	backward.Add(from)

	// Add empty value to make sure from has a backward edge.
	if _, ok := g.backwardEdges[from]; !ok {
		g.backwardEdges[from] = NewStringSet()
	}
	if _, ok := g.forwardEdges[to]; !ok {
		g.forwardEdges[to] = NewStringSet()
	}

	return nil
}

type Iterator struct {
	sync.Mutex

	graph   *Graph
	next    *StringSet
	visited map[string]bool
	deps    map[string]*StringSet
}

func InitIterator(g *Graph) *Iterator {
	next := NewStringSet()
	deps := map[string]*StringSet{}
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

	iter.next = NewStringSet()

	return nextNodes
}

func (iter *Iterator) Start(node string) error {
	if iter.Visited(node) {
		return fmt.Errorf("%q already visited", node)
	}

	iter.visitNode(node)

	forward := iter.forwardEdges(node)

	if forward.Len() == 0 {
		iter.addNext(node)
		return nil
	}

	for _, e := range forward.Values() {
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

func (iter *Iterator) forwardEdges(node string) *StringSet {
	iter.Lock()
	defer iter.Unlock()

	return iter.graph.forwardEdges[node]
}

func (iter *Iterator) backwardEdges(node string) *StringSet {
	iter.Lock()
	defer iter.Unlock()

	return iter.graph.backwardEdges[node]
}

func (iter *Iterator) visitNode(node string) {
	iter.Lock()
	defer iter.Unlock()

	iter.visited[node] = true
}

func (iter *Iterator) dependencies(node string) *StringSet {
	iter.Lock()
	defer iter.Unlock()

	deps, ok := iter.deps[node]
	if !ok {
		deps = NewStringSet()
		iter.deps[node] = deps
	}

	return deps
}

func NewStringSet() *StringSet {
	return &StringSet{
		values: map[string]struct{}{},
	}
}

type StringSet struct {
	sync.Mutex

	values map[string]struct{}
}

func (s *StringSet) Add(val string) {
	s.Lock()
	defer s.Unlock()

	s.values[val] = struct{}{}
}

func (s *StringSet) Remove(val string) {
	s.Lock()
	defer s.Unlock()

	delete(s.values, val)
}

func (s *StringSet) Clone() *StringSet {
	s.Lock()
	defer s.Unlock()

	dest := NewStringSet()
	for k := range s.values {
		dest.Add(k)
	}

	return dest
}

func (s *StringSet) Len() int {
	s.Lock()
	defer s.Unlock()

	return len(s.values)
}

func (s *StringSet) Values() []string {
	s.Lock()
	defer s.Unlock()

	vals := make([]string, 0, len(s.values))
	for v := range s.values {
		vals = append(vals, v)
	}

	return vals
}
