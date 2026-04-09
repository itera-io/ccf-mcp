package main

import "sort"

func applyOffsetLimit[T any](items []T, offset, limit int32) []T {
	if offset < 0 {
		offset = 0
	}
	if limit < 0 {
		limit = 0
	}

	start := int(offset)
	if start > len(items) {
		start = len(items)
	}

	end := len(items)
	if limit > 0 && start+int(limit) < end {
		end = start + int(limit)
	}

	return items[start:end]
}

func sortAndPaginateStrings(values []string, offset, limit int32) []string {
	sorted := append([]string(nil), values...)
	sort.Strings(sorted)
	return applyOffsetLimit(sorted, offset, limit)
}
