import "@xyflow/react/dist/style.css";

import dagre from "dagre";
import {
  Background,
  Controls,
  MiniMap,
  ReactFlow,
  ReactFlowProvider,
  useEdgesState,
  useNodesState,
  useReactFlow,
  type Edge,
  type Node
} from "@xyflow/react";
import { useEffect, useMemo, useState } from "react";

import type { WorkflowState, WorkflowTask } from "../types/api";

type DagViewProps = {
  tasks: WorkflowTask[];
};

type TaskNodeData = {
  label: string;
  status: WorkflowState;
};

const NODE_WIDTH = 220;
const NODE_HEIGHT = 72;

function nodeClass(status: WorkflowState) {
  switch (status) {
    case "running":
      return "border-blue-400 bg-blue-50 dark:border-blue-500 dark:bg-blue-900/40";
    case "completed":
      return "border-emerald-400 bg-emerald-50 dark:border-emerald-500 dark:bg-emerald-900/40";
    case "failed":
      return "border-red-400 bg-red-50 dark:border-red-500 dark:bg-red-900/40";
    case "cancelled":
      return "border-amber-400 bg-amber-50 dark:border-amber-500 dark:bg-amber-900/40";
    case "pending":
      return "border-dashed border-zinc-400 bg-zinc-50 dark:border-zinc-500 dark:bg-zinc-900/30";
    case "scheduled":
    default:
      return "border-zinc-400 bg-zinc-50 dark:border-zinc-500 dark:bg-zinc-900/40";
  }
}

function statusIcon(status: WorkflowState) {
  switch (status) {
    case "running":
      return "o";
    case "completed":
      return "v";
    case "failed":
      return "x";
    case "cancelled":
      return "!";
    case "pending":
      return "...";
    case "scheduled":
    default:
      return "~";
  }
}

function createLayout(tasks: WorkflowTask[]) {
  const hasDependencies = tasks.some((task) => (task.depends_on?.length ?? 0) > 0);

  const nodes: Node<TaskNodeData>[] = tasks.map((task, index) => ({
    id: task.id,
    type: "default",
    data: { label: task.name, status: task.status },
    position: hasDependencies
      ? { x: 0, y: 0 }
      : {
          x: index * (NODE_WIDTH + 40),
          y: 80
        },
    style: {
      width: NODE_WIDTH,
      height: NODE_HEIGHT
    }
  }));

  const edges: Edge[] = [];
  tasks.forEach((task) => {
    for (const dep of task.depends_on ?? []) {
      edges.push({
        id: `${dep}->${task.id}`,
        source: dep,
        target: task.id,
        animated: task.status === "running",
        markerEnd: { type: "arrowclosed" }
      });
    }
  });

  if (!hasDependencies) {
    return { nodes, edges };
  }

  const graph = new dagre.graphlib.Graph();
  graph.setGraph({ rankdir: "TB", nodesep: 40, ranksep: 72 });
  graph.setDefaultEdgeLabel(() => ({}));

  nodes.forEach((node) => {
    graph.setNode(node.id, { width: NODE_WIDTH, height: NODE_HEIGHT });
  });
  edges.forEach((edge) => {
    graph.setEdge(edge.source, edge.target);
  });

  dagre.layout(graph);

  const laidOutNodes = nodes.map((node) => {
    const position = graph.node(node.id);
    return {
      ...node,
      position: {
        x: position.x - NODE_WIDTH / 2,
        y: position.y - NODE_HEIGHT / 2
      }
    };
  });

  return { nodes: laidOutNodes, edges };
}

function styleForEdge(edge: Edge, taskByID: Record<string, WorkflowTask>) {
  const target = taskByID[edge.target];
  const source = taskByID[edge.source];
  const completed = source?.status === "completed" && target?.status === "completed";
  return {
    ...edge,
    style: completed ? { stroke: "#21a56b", strokeWidth: 2 } : { stroke: "#8b97a2", strokeDasharray: "6 4" }
  };
}

function DagCanvas({ tasks }: DagViewProps) {
  const taskByID = useMemo<Record<string, WorkflowTask>>(
    () =>
      tasks.reduce<Record<string, WorkflowTask>>((acc, item) => {
        acc[item.id] = item;
        return acc;
      }, {}),
    [tasks]
  );
  const signature = useMemo(
    () => tasks.map((task) => `${task.id}:${(task.depends_on ?? []).join(",")}`).join("|"),
    [tasks]
  );
  const statusSignature = useMemo(
    () => tasks.map((task) => `${task.id}:${task.status}`).join("|"),
    [tasks]
  );
  const [selectedNodeID, setSelectedNodeID] = useState<string | null>(null);
  const [nodes, setNodes, onNodesChange] = useNodesState<Node<TaskNodeData>>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const reactFlow = useReactFlow();

  useEffect(() => {
    const { nodes: nextNodes, edges: nextEdges } = createLayout(tasks);
    setNodes(
      nextNodes.map((node) => ({
        ...node,
        className: `rounded-lg border ${nodeClass(taskByID[node.id]?.status ?? "pending")}`
      }))
    );
    setEdges(nextEdges.map((edge) => styleForEdge(edge, taskByID)));
  }, [signature, setEdges, setNodes, tasks, taskByID]);

  useEffect(() => {
    setNodes((current) =>
      current.map((node) => ({
        ...node,
        data: {
          ...node.data,
          status: taskByID[node.id]?.status ?? node.data.status
        },
        className: `rounded-lg border ${nodeClass(taskByID[node.id]?.status ?? "pending")}`
      }))
    );
    setEdges((current) => current.map((edge) => styleForEdge(edge, taskByID)));
  }, [setEdges, setNodes, statusSignature, taskByID]);

  useEffect(() => {
    if (nodes.length > 0) {
      void reactFlow.fitView({ padding: 0.2 });
    }
  }, [nodes.length, reactFlow]);

  const selectedTask = selectedNodeID ? taskByID[selectedNodeID] : null;

  return (
    <div className="grid gap-4 lg:grid-cols-[1fr_320px]">
      <div className="relative h-[560px] rounded-xl border border-[var(--ui-border)] bg-[var(--ui-panel)]">
        <button
          type="button"
          onClick={() => void reactFlow.fitView({ padding: 0.2 })}
          className="absolute right-3 top-3 z-10 rounded-md border border-[var(--ui-border)] bg-[var(--ui-panel)] px-3 py-1 text-xs font-semibold"
        >
          Fit to View
        </button>

        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onNodeClick={(_, node) => setSelectedNodeID(node.id)}
          fitView
          fitViewOptions={{ padding: 0.2 }}
          nodesConnectable={false}
        >
          <Background />
          <Controls />
          {nodes.length > 10 ? <MiniMap pannable zoomable /> : null}
        </ReactFlow>
      </div>

      <aside className="rounded-xl border border-[var(--ui-border)] bg-[var(--ui-panel)] p-4">
        <h3 className="text-sm font-semibold uppercase tracking-wide text-[var(--ui-muted)]">
          Task Details
        </h3>
        {!selectedTask ? (
          <p className="mt-3 text-sm text-[var(--ui-muted)]">Click a node to inspect task details.</p>
        ) : (
          <div className="mt-3 space-y-3 text-sm">
            <p className="font-semibold">{selectedTask.name}</p>
            <p className="font-mono text-xs text-[var(--ui-muted)]">{selectedTask.id}</p>
            <p className="inline-flex items-center gap-2 rounded border border-[var(--ui-border)] px-2 py-1 text-xs">
              <span>{statusIcon(selectedTask.status)}</span>
              <span className="capitalize">{selectedTask.status}</span>
            </p>
            <p className="text-xs text-[var(--ui-muted)]">
              Dependencies: {(selectedTask.depends_on ?? []).length === 0 ? "None" : (selectedTask.depends_on ?? []).join(", ")}
            </p>
            {selectedTask.error ? (
              <p className="rounded border border-red-300/50 bg-red-50/60 p-2 text-xs text-red-700 dark:border-red-500/50 dark:bg-red-900/20 dark:text-red-200">
                {selectedTask.error}
              </p>
            ) : null}
          </div>
        )}
      </aside>
    </div>
  );
}

export function DagView(props: DagViewProps) {
  return (
    <ReactFlowProvider>
      <DagCanvas {...props} />
    </ReactFlowProvider>
  );
}

