package gatewaysvc

import (
	"sort"
	"strconv"
)

func tcpEntrypointName(listen int) string { return "tcp" + strconv.Itoa(listen) }

func normalizeTCPListens(listens []int) []int {
	seen := map[int]struct{}{}
	result := make([]int, 0, len(listens))
	for _, listen := range listens {
		if listen < 1 || listen > 65535 || listen == 80 || listen == 443 || listen == 8080 {
			continue
		}
		if _, exists := seen[listen]; exists {
			continue
		}
		seen[listen] = struct{}{}
		result = append(result, listen)
	}
	sort.Ints(result)
	return result
}
