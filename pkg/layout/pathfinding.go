package layout

import (
	"container/heap"
	"fmt"
)

// Path represents a sequence of grid coordinates.
type Path []Coord

// FindPath uses A* pathfinding to find a path between two coordinates on a grid.
func FindPath(grid *Grid, from, to Coord) (Path, error) {
	pq := &pqueue{}
	heap.Init(pq)
	heap.Push(pq, &pqItem{coord: from, priority: 0})

	costSoFar := map[Coord]int{from: 0}
	cameFrom := map[Coord]*Coord{from: nil}

	directions := []Coord{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

	for pq.Len() > 0 {
		current := heap.Pop(pq).(*pqItem).coord

		if current == to {
			var path Path
			for c := &current; c != nil; c = cameFrom[*c] {
				path = append(Path{*c}, path...)
			}
			return path, nil
		}

		for _, dir := range directions {
			next := Coord{X: current.X + dir.X, Y: current.Y + dir.Y}
			if next.X < 0 || next.Y < 0 {
				continue
			}
			if grid.IsOccupied(next) && next != to {
				continue
			}

			newCost := costSoFar[current] + 1
			if cost, ok := costSoFar[next]; !ok || newCost < cost {
				costSoFar[next] = newCost
				priority := newCost + heuristic(next, to)
				heap.Push(pq, &pqItem{coord: next, priority: priority})
				cameFrom[next] = &current
			}
		}
	}
	return nil, fmt.Errorf("no path found from %v to %v", from, to)
}

func heuristic(a, b Coord) int {
	absX := a.X - b.X
	if absX < 0 {
		absX = -absX
	}
	absY := a.Y - b.Y
	if absY < 0 {
		absY = -absY
	}
	if absX == 0 || absY == 0 {
		return absX + absY
	}
	return absX + absY + 1 // Penalty for corners
}

// Priority queue implementation
type pqItem struct {
	coord    Coord
	priority int
	index    int
}

type pqueue []*pqItem

func (pq pqueue) Len() int            { return len(pq) }
func (pq pqueue) Less(i, j int) bool  { return pq[i].priority < pq[j].priority }
func (pq pqueue) Swap(i, j int)       { pq[i], pq[j] = pq[j], pq[i]; pq[i].index = i; pq[j].index = j }
func (pq *pqueue) Push(x interface{}) { n := len(*pq); item := x.(*pqItem); item.index = n; *pq = append(*pq, item) }
func (pq *pqueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[:n-1]
	return item
}
