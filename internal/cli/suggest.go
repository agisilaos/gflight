package cli

import "strings"

func suggestClosest(input string, choices []string) string {
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" || len(choices) == 0 {
		return ""
	}
	best := ""
	bestDist := 1 << 30
	for _, c := range choices {
		cn := strings.ToLower(c)
		if cn == input {
			return c
		}
		d := levenshtein(input, cn)
		if d < bestDist {
			bestDist = d
			best = c
		}
	}
	limit := 2
	if len(input) >= 8 {
		limit = 3
	}
	if bestDist <= limit {
		return best
	}
	return ""
}

func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	if a == "" {
		return len(b)
	}
	if b == "" {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := 0; j <= len(b); j++ {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			curr[j] = min3(del, ins, sub)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

func min3(a, b, c int) int {
	if a <= b && a <= c {
		return a
	}
	if b <= a && b <= c {
		return b
	}
	return c
}
