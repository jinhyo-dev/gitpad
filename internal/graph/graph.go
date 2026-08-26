// Package graph lays out a commit DAG into text lanes, one row per commit.
package graph

// Node is the minimal commit information needed for layout.
type Node struct {
	Hash    string
	Parents []string
}

// Cell is one terminal column of a row: a glyph and the lane color index.
type Cell struct {
	Rune  rune
	Color int // palette index, -1 for none
}

// Row is the rendered graph for one commit.
type Row struct {
	Cells    []Cell
	NodeLane int // lane index of the commit node
	NodeCol  int // column (cell index) of the node glyph
	IsMerge  bool
}

const (
	glyphNode  = '●'
	glyphMerge = '○'
)

type lane struct {
	hash   string
	color  int
	active bool
}

// Build lays out nodes, which must be in an order where every commit precedes
// its parents (git log --date-order / --topo-order guarantee this).
func Build(nodes []Node, palette int) []Row {
	if palette <= 0 {
		palette = 1
	}
	var lanes []lane
	nextColor := 0
	alloc := func(hash string) int {
		for j := range lanes {
			if !lanes[j].active {
				lanes[j] = lane{hash: hash, color: nextColor % palette, active: true}
				nextColor++
				return j
			}
		}
		lanes = append(lanes, lane{hash: hash, color: nextColor % palette, active: true})
		nextColor++
		return len(lanes) - 1
	}
	rows := make([]Row, len(nodes))
	for idx, n := range nodes {
		// Which lane carries this commit?
		i := -1
		var dups []int
		for j, l := range lanes {
			if l.active && l.hash == n.Hash {
				if i == -1 {
					i = j
				} else {
					dups = append(dups, j)
				}
			}
		}
		if i == -1 {
			i = alloc(n.Hash)
		}
		nodeColor := lanes[i].color

		var newLanes, joinLanes []int
		if len(n.Parents) == 0 {
			lanes[i].active = false // closed after rendering; keep color for the node
		}
		for pi, p := range n.Parents {
			if pi == 0 {
				lanes[i].hash = p
				continue
			}
			k := -1
			for j, l := range lanes {
				if j != i && l.active && l.hash == p && !contains(dups, j) {
					k = j
					break
				}
			}
			if k >= 0 {
				joinLanes = append(joinLanes, k)
			} else {
				newLanes = append(newLanes, alloc(p))
			}
		}

		lo, hi := i, i
		for _, s := range [][]int{dups, newLanes, joinLanes} {
			for _, j := range s {
				if j < lo {
					lo = j
				}
				if j > hi {
					hi = j
				}
			}
		}
		// Visible width: last lane that is active or part of this row's shape.
		width := hi + 1
		for j := len(lanes) - 1; j >= width; j-- {
			if lanes[j].active {
				width = j + 1
				break
			}
		}
		cells := make([]Cell, 0, width*2)
		row := Row{NodeLane: i, IsMerge: len(n.Parents) > 1}
		for j := 0; j < width; j++ {
			l := lanes[j]
			var g rune
			color := l.color
			between := lo < j && j < hi
			switch {
			case j == i:
				g = glyphNode
				if row.IsMerge {
					g = glyphMerge
				}
				color = nodeColor
				row.NodeCol = len(cells)
			case contains(dups, j):
				if j > i {
					g = '╯'
				} else {
					g = '╰'
				}
			case contains(newLanes, j):
				if j > i {
					g = '╮'
				} else {
					g = '╭'
				}
			case contains(joinLanes, j):
				if j > i {
					g = '┤'
				} else {
					g = '├'
				}
			case l.active && (j != i):
				if between {
					g = '┼'
				} else {
					g = '│'
				}
			default:
				if between {
					g = '─'
					color = nodeColor
				} else {
					g = ' '
					color = -1
				}
			}
			cells = append(cells, Cell{Rune: g, Color: color})
			if j < width-1 {
				if lo <= j && j < hi {
					cells = append(cells, Cell{Rune: '─', Color: nodeColor})
				} else {
					cells = append(cells, Cell{Rune: ' ', Color: -1})
				}
			}
		}
		row.Cells = cells
		rows[idx] = row

		for _, j := range dups {
			lanes[j].active = false
		}
		for len(lanes) > 0 && !lanes[len(lanes)-1].active {
			lanes = lanes[:len(lanes)-1]
		}
	}
	return rows
}

func contains(s []int, v int) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
