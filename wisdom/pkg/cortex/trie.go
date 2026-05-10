package cortex

import (
	"strings"
	"sync"
)

// TrieNode represents a single character/token in the SCG-Mem Prefix Trie.
type TrieNode struct {
	Children map[rune]*TrieNode
	IsEnd    bool
	NodeIDs  []string // IDs of nodes that contain this exact prefix
}

// SCGTrie provides Source-Constrained Generation validation.
type SCGTrie struct {
	Root *TrieNode
	mu   sync.RWMutex
}

// NewSCGTrie initializes an empty trie.
func NewSCGTrie() *SCGTrie {
	return &SCGTrie{
		Root: &TrieNode{Children: make(map[rune]*TrieNode)},
	}
}

// Insert adds a string (and its associated node ID) to the trie.
func (t *SCGTrie) Insert(text string, nodeID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// We index words or phrases to keep it efficient
	words := strings.Fields(strings.ToLower(text))
	for _, word := range words {
		curr := t.Root
		for _, char := range word {
			if _, ok := curr.Children[char]; !ok {
				curr.Children[char] = &TrieNode{Children: make(map[rune]*TrieNode)}
			}
			curr = curr.Children[char]
			// Keep track of which nodes contribute to this prefix path
			found := false
			for _, id := range curr.NodeIDs {
				if id == nodeID {
					found = true
					break
				}
			}
			if !found {
				curr.NodeIDs = append(curr.NodeIDs, nodeID)
			}
		}
		curr.IsEnd = true
	}
}

// Exists checks if a word or prefix exists in the trie.
func (t *SCGTrie) Exists(word string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	curr := t.Root
	for _, char := range strings.ToLower(word) {
		if _, ok := curr.Children[char]; !ok {
			return false
		}
		curr = curr.Children[char]
	}
	return curr.IsEnd
}

// ValidateSentence checks how many words in a sentence are "grounded" in the trie.
func (t *SCGTrie) ValidateSentence(sentence string) (groundedCount int, ungrounded []string) {
	words := strings.Fields(strings.ToLower(sentence))
	for _, word := range words {
		// Clean punctuation
		clean := strings.Trim(word, ".,!?;:()\"")
		if clean == "" || len(clean) < 3 { continue } // Skip short stop words
		
		if t.Exists(clean) {
			groundedCount++
		} else {
			ungrounded = append(ungrounded, clean)
		}
	}
	return groundedCount, ungrounded
}
