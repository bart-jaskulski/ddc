package main

import (
	"strconv"
	"strings"
	"unicode"
)

// versionParts splits a version string into comparable parts
func versionParts(v string) []string {
	var parts []string
	current := ""

	for _, c := range v {
		if len(current) == 0 {
			current = string(c)
			continue
		}

		// Check if we're switching between digit and non-digit
		currentIsDigit := unicode.IsDigit(rune(current[len(current)-1]))
		newIsDigit := unicode.IsDigit(c)

		if currentIsDigit == newIsDigit {
			current += string(c)
		} else {
			parts = append(parts, current)
			current = string(c)
		}
	}

	if len(current) > 0 {
		parts = append(parts, current)
	}

	return parts
}

// compareVersionParts compares two version parts naturally
func compareVersionParts(a, b string) int {
	// If both are numbers, compare numerically
	aNum, aErr := strconv.Atoi(a)
	bNum, bErr := strconv.Atoi(b)
	if aErr == nil && bErr == nil {
		if aNum == bNum {
			return 0
		}
		if aNum > bNum {
			return 1
		}
		return -1
	}

	// If one is a number and one isn't, number comes first
	if aErr == nil {
		return -1
	}
	if bErr == nil {
		return 1
	}

	// Otherwise compare strings
	return strings.Compare(a, b)
}

// CompareVersions compares two version strings naturally
// Returns -1 if a < b, 0 if a == b, 1 if a > b
func CompareVersions(a, b string) int {
	aParts := versionParts(a)
	bParts := versionParts(b)

	for i := 0; i < len(aParts) && i < len(bParts); i++ {
		if cmp := compareVersionParts(aParts[i], bParts[i]); cmp != 0 {
			return cmp
		}
	}

	// If all parts are equal, longer version is greater
	if len(aParts) > len(bParts) {
		return 1
	}
	if len(aParts) < len(bParts) {
		return -1
	}
	return 0
}
