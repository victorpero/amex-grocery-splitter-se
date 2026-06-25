package matcher

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

var defaultStorePrefixes = []string{
	"HEMKOP",
	"ICA",
	"MAXI ICA",
	"WILLYS",
	"COOP",
	"PRESSBYRÅN",
}

type PrefixMatcher struct {
	prefixes []storePrefix
}

type storePrefix struct {
	normalized string
	folded     string
}

func DefaultStorePrefixes() []string {
	prefixes := make([]string, len(defaultStorePrefixes))
	copy(prefixes, defaultStorePrefixes)
	return prefixes
}

func NewPrefixMatcher(prefixes []string) (*PrefixMatcher, error) {
	if len(prefixes) == 0 {
		return nil, fmt.Errorf("at least one grocery store prefix is required")
	}

	seen := make(map[string]struct{}, len(prefixes))
	normalizedPrefixes := make([]storePrefix, 0, len(prefixes))
	for _, prefix := range prefixes {
		normalized := NormalizeDescription(prefix)
		if normalized == "" {
			continue
		}
		folded := FoldSwedish(normalized)
		if _, exists := seen[folded]; exists {
			continue
		}
		seen[folded] = struct{}{}
		normalizedPrefixes = append(normalizedPrefixes, storePrefix{
			normalized: normalized,
			folded:     folded,
		})
	}

	if len(normalizedPrefixes) == 0 {
		return nil, fmt.Errorf("at least one non-empty grocery store prefix is required")
	}

	return &PrefixMatcher{prefixes: normalizedPrefixes}, nil
}

func (m *PrefixMatcher) IsGrocery(description string) bool {
	normalized := NormalizeDescription(description)
	if normalized == "" {
		return false
	}

	folded := FoldSwedish(normalized)
	for _, prefix := range m.prefixes {
		if strings.HasPrefix(normalized, prefix.normalized) || strings.HasPrefix(folded, prefix.folded) {
			return true
		}
	}
	return false
}

func NormalizeDescription(description string) string {
	return strings.ToUpper(strings.Join(strings.Fields(strings.TrimSpace(description)), " "))
}

func FoldSwedish(value string) string {
	replacer := strings.NewReplacer(
		"Å", "A",
		"Ä", "A",
		"Ö", "O",
		"å", "A",
		"ä", "A",
		"ö", "O",
	)
	return replacer.Replace(value)
}

func LoadPrefixesFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open grocery store prefix file: %w", err)
	}
	defer file.Close()

	prefixes, err := LoadPrefixes(file)
	if err != nil {
		return nil, fmt.Errorf("read grocery store prefix file: %w", err)
	}
	if len(prefixes) == 0 {
		return nil, fmt.Errorf("grocery store prefix file %q did not contain any prefixes", path)
	}
	return prefixes, nil
}

func LoadPrefixes(reader io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(reader)
	prefixes := make([]string, 0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		prefixes = append(prefixes, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return prefixes, nil
}
