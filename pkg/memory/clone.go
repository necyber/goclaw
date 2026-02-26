package memory

func cloneEntry(entry *MemoryEntry) *MemoryEntry {
	if entry == nil {
		return nil
	}
	clone := *entry
	if entry.Vector != nil {
		clone.Vector = append([]float32(nil), entry.Vector...)
	}
	if entry.Metadata != nil {
		clone.Metadata = make(map[string]string, len(entry.Metadata))
		for key, value := range entry.Metadata {
			clone.Metadata[key] = value
		}
	}
	return &clone
}
