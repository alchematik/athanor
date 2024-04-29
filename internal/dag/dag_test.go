package dag_test

import (
	"testing"

	"github.com/alchematik/athanor/internal/dag"
	"github.com/stretchr/testify/require"
)

func TestIterator(t *testing.T) {
	/*

		a ---> b ---> c

	*/
	g := dag.NewGraph()
	require.NoError(t, g.AddEdge("a", "b"))
	require.NoError(t, g.AddEdge("b", "c"))

	iter := dag.InitIterator(g)
	next := iter.Next()
	var result []string
	for len(next) > 0 {
		for _, n := range next {
			result = append(result, n)
			if iter.Visited(n) {
				require.NoError(t, iter.Done(n))
			} else {
				require.NoError(t, iter.Start(n))
			}
		}
		next = iter.Next()
	}

	require.Equal(t, []string{"a", "b", "c", "c", "b", "a"}, result)

	/*
	   a
	   |---> b
	   └---> c
	*/
	g = dag.NewGraph()
	require.NoError(t, g.AddEdge("a", "b"))
	require.NoError(t, g.AddEdge("a", "c"))

	iter = dag.InitIterator(g)
	next = iter.Next()
	require.False(t, iter.Visited("a"))
	require.Equal(t, []string{"a"}, next)
	require.NoError(t, iter.Start("a"))

	next = iter.Next()
	require.ElementsMatch(t, []string{"b", "c"}, next)
	require.False(t, iter.Visited("b"))
	require.False(t, iter.Visited("c"))
	require.NoError(t, iter.Start("b"))
	require.NoError(t, iter.Start("c"))

	next = iter.Next()
	require.ElementsMatch(t, []string{"b", "c"}, next)
	require.True(t, iter.Visited("b"))
	require.True(t, iter.Visited("c"))
	require.NoError(t, iter.Done("b"))
	require.NoError(t, iter.Done("c"))

	next = iter.Next()
	require.Equal(t, []string{"a"}, next)
	require.True(t, iter.Visited("a"))
	require.NoError(t, iter.Done("a"))

	/*
	   a
	   |--------> b ---> e
	   |---> c ---^
	   └---> d ---> f
	*/

	g = dag.NewGraph()
	require.NoError(t, g.AddEdge("a", "b"))
	require.NoError(t, g.AddEdge("b", "e"))
	require.NoError(t, g.AddEdge("a", "c"))
	require.NoError(t, g.AddEdge("c", "b"))
	require.NoError(t, g.AddEdge("a", "d"))
	require.NoError(t, g.AddEdge("d", "f"))

	iter = dag.InitIterator(g)
	next = iter.Next()
	require.Equal(t, []string{"a"}, next)
	require.False(t, iter.Visited("a"))
	require.NoError(t, iter.Start("a"))

	next = iter.Next()
	require.ElementsMatch(t, []string{"c", "d"}, next)
	require.False(t, iter.Visited("c"))
	require.False(t, iter.Visited("d"))
	require.NoError(t, iter.Start("c"))
	require.NoError(t, iter.Start("d"))

	next = iter.Next()
	require.ElementsMatch(t, []string{"b", "f"}, next)
	require.False(t, iter.Visited("b"))
	require.False(t, iter.Visited("f"))
	require.NoError(t, iter.Start("b"))
	require.NoError(t, iter.Start("f"))

	next = iter.Next()
	require.ElementsMatch(t, []string{"e", "f"}, next)
	require.False(t, iter.Visited("e"))
	require.True(t, iter.Visited("f"))
	require.NoError(t, iter.Start("e"))
	require.NoError(t, iter.Done("f"))

	next = iter.Next()
	require.ElementsMatch(t, []string{"e", "d"}, next)
	require.True(t, iter.Visited("e"))
	require.True(t, iter.Visited("d"))
	require.NoError(t, iter.Done("e"))
	require.NoError(t, iter.Done("d"))

	next = iter.Next()
	require.ElementsMatch(t, []string{"b"}, next)
	require.True(t, iter.Visited("b"))
	require.NoError(t, iter.Done("b"))

	next = iter.Next()
	require.ElementsMatch(t, []string{"c"}, next)
	require.True(t, iter.Visited("c"))
	require.NoError(t, iter.Done("c"))

	next = iter.Next()
	require.ElementsMatch(t, []string{"a"}, next)
	require.True(t, iter.Visited("a"))
	require.NoError(t, iter.Done("a"))
}

func TestIteratorConcurrent(t *testing.T) {
	/*
	   a
	   |--------> b ---> e
	   |---> c ---^
	   └---> d ---> f
	*/

	g := dag.NewGraph()
	require.NoError(t, g.AddEdge("a", "b"))
	require.NoError(t, g.AddEdge("b", "e"))
	require.NoError(t, g.AddEdge("a", "c"))
	require.NoError(t, g.AddEdge("c", "b"))
	require.NoError(t, g.AddEdge("a", "d"))
	require.NoError(t, g.AddEdge("d", "f"))

	iter := dag.InitIterator(g)
	ch := make(chan string)
	done := make(chan any)
	go func() {
		for n := range ch {
			n := n
			go func() {
				if iter.Visited(n) {
					if n == "a" {
						close(ch)
						close(done)
					}
					require.NoError(t, iter.Done(n))
				} else {
					require.NoError(t, iter.Start(n))
				}
				next := iter.Next()
				for _, nx := range next {
					ch <- nx
				}
			}()
		}
	}()

	for _, n := range iter.Next() {
		ch <- n
	}

	<-done
}
