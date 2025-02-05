package main

type DocumentEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

// GetDisplayName returns a formatted name for display in the UI
func (d *Documentation) GetDisplayName() string {
	if d.Version != "" {
		return d.Name + " " + d.Version
	}
	return d.Name
}

// SplitFragment splits a path into base path and fragment
func (e *DocumentEntry) SplitFragment() (string, string) {
	for i := 0; i < len(e.Path); i++ {
		if e.Path[i] == '#' {
			return e.Path[:i], e.Path[i:]
		}
	}
	return e.Path, ""
}

// GetDisplayEntry returns a formatted string for UI display
func (e *DocumentEntry) GetDisplayEntry() string {
	return e.Name + " (" + e.Type + ")"
}
