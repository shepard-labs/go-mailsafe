package emailverifier

import "strings"

func (v *Verifier) SuggestDomain(domain string) string {
	domain = strings.ToLower(domain)

	for _, known := range suggestDomainList {
		if known == domain {
			return ""
		}
	}

	var bestMatch string
	bestDist := 3 // max edit distance threshold

	for _, known := range suggestDomainList {
		dist := levenshtein(domain, known)
		if dist < bestDist {
			bestDist = dist
			bestMatch = known
		}
	}

	return bestMatch
}

func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(
				prev[j]+1,
				curr[j-1]+1,
				prev[j-1]+cost,
			)
		}
		prev, curr = curr, prev
	}

	return prev[lb]
}

func min(vals ...int) int {
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}
