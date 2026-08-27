// Port of internal/graph: lays a commit DAG into lanes, one row per commit.
export type Node = { hash: string; parents: string[] };
export type Cell = { ch: string; color: number };
export type Row = { cells: Cell[]; nodeCol: number; isMerge: boolean };

type Lane = { hash: string; color: number; active: boolean };

export function buildGraph(nodes: Node[], palette: number): Row[] {
  let lanes: Lane[] = [];
  let nextColor = 0;
  const alloc = (hash: string): number => {
    for (let j = 0; j < lanes.length; j++) {
      if (!lanes[j].active) {
        lanes[j] = { hash, color: nextColor++ % palette, active: true };
        return j;
      }
    }
    lanes.push({ hash, color: nextColor++ % palette, active: true });
    return lanes.length - 1;
  };
  const rows: Row[] = [];
  for (const n of nodes) {
    let i = -1;
    const dups: number[] = [];
    lanes.forEach((l, j) => {
      if (l.active && l.hash === n.hash) {
        if (i === -1) i = j;
        else dups.push(j);
      }
    });
    if (i === -1) i = alloc(n.hash);
    const nodeColor = lanes[i].color;
    const newLanes: number[] = [];
    const joinLanes: number[] = [];
    if (n.parents.length === 0) lanes[i].active = false;
    n.parents.forEach((p, pi) => {
      if (pi === 0) {
        lanes[i].hash = p;
        return;
      }
      const k = lanes.findIndex((l, j) => j !== i && l.active && l.hash === p && !dups.includes(j));
      if (k >= 0) joinLanes.push(k);
      else newLanes.push(alloc(p));
    });
    let lo = i;
    let hi = i;
    for (const j of [...dups, ...newLanes, ...joinLanes]) {
      lo = Math.min(lo, j);
      hi = Math.max(hi, j);
    }
    let width = hi + 1;
    for (let j = lanes.length - 1; j >= width; j--) {
      if (lanes[j].active) {
        width = j + 1;
        break;
      }
    }
    const cells: Cell[] = [];
    const row: Row = { cells, nodeCol: 0, isMerge: n.parents.length > 1 };
    for (let j = 0; j < width; j++) {
      const l = lanes[j];
      const between = lo < j && j < hi;
      let ch: string;
      let color = l.color;
      if (j === i) {
        ch = row.isMerge ? "○" : "●";
        color = nodeColor;
        row.nodeCol = cells.length;
      } else if (dups.includes(j)) ch = j > i ? "╯" : "╰";
      else if (newLanes.includes(j)) ch = j > i ? "╮" : "╭";
      else if (joinLanes.includes(j)) ch = j > i ? "┤" : "├";
      else if (l.active) ch = between ? "┼" : "│";
      else {
        ch = between ? "─" : " ";
        color = between ? nodeColor : -1;
      }
      cells.push({ ch, color });
      if (j < width - 1)
        cells.push(lo <= j && j < hi ? { ch: "─", color: nodeColor } : { ch: " ", color: -1 });
    }
    rows.push(row);
    for (const j of dups) lanes[j].active = false;
    while (lanes.length > 0 && !lanes[lanes.length - 1].active) lanes.pop();
  }
  return rows;
}
