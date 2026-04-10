interface NodeWithDeps {
	name: string
	depends_on: string[]
}

/**
 * 检测节点列表中是否存在循环依赖（DFS）
 * @returns 存在循环时返回完整环路径（name），否则返回 null
 */
export function detectCircularDependencies(stages: NodeWithDeps[]): string[] | null {
	const graph = new Map<string, string[]>();
	for (const stage of stages) {
		graph.set(stage.name, stage.depends_on);
	}

	const visited = new Set<string>();
	const recStack = new Set<string>();

	function dfs(node: string, path: string[]): string[] | null {
		if (recStack.has(node)) {
			const cycleStart = path.indexOf(node);
			return [...path.slice(cycleStart), node];
		}
		if (visited.has(node)) {
			return null;
		}

		visited.add(node);
		recStack.add(node);

		for (const dep of graph.get(node) || []) {
			const cycle = dfs(dep, [...path, node]);
			if (cycle) {
				return cycle;
			}
		}

		recStack.delete(node);
		return null;
	}

	for (const stage of stages) {
		const cycle = dfs(stage.name, []);
		if (cycle) {
			return cycle;
		}
	}

	return null;
}

export function hasCycle(stages: NodeWithDeps[]): boolean {
	return detectCircularDependencies(stages) !== null;
}
