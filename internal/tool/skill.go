package tool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SkillTool loads a specialized skill that provides domain-specific instructions
// Mirrors opencode's skill tool
func SkillTool() *ToolDef {
	return &ToolDef{
		Name:        "skill",
		Description: "Load a specialized skill from .dcode/skills/ for domain-specific workflows.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "The name of the skill to load",
				},
			},
			"required": []string{"name"},
		},
		Execute: func(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
			name, _ := input["name"].(string)
			if name == "" {
				return &ToolResult{Output: "Error: skill name is required", IsError: true}, nil
			}

			// Search for skill files in multiple locations
			searchPaths := []string{
				filepath.Join(tc.WorkDir, ".dcode", "skills", name+".md"),
				filepath.Join(tc.WorkDir, ".dcode", "skills", name),
				filepath.Join(tc.WorkDir, ".opencode", "skills", name+".md"),
				filepath.Join(tc.WorkDir, ".opencode", "skills", name),
			}

			// Also check config dir
			home, _ := os.UserHomeDir()
			if home != "" {
				searchPaths = append(searchPaths,
					filepath.Join(home, ".config", "dcode", "skills", name+".md"),
					filepath.Join(home, ".config", "dcode", "skills", name),
				)
			}

			for _, path := range searchPaths {
				data, err := os.ReadFile(path)
				if err == nil {
					return &ToolResult{
						Output: fmt.Sprintf("Loaded skill '%s':\n\n%s", name, string(data)),
					}, nil
				}
			}

			// List available skills
			available := listAvailableSkills(tc.WorkDir)
			if len(available) > 0 {
				return &ToolResult{
					Output:  fmt.Sprintf("Skill '%s' not found. Available skills: %s", name, strings.Join(available, ", ")),
					IsError: true,
				}, nil
			}

			return &ToolResult{
				Output:  fmt.Sprintf("Skill '%s' not found. No skills are currently available. Create skills in .dcode/skills/ as markdown files.", name),
				IsError: true,
			}, nil
		},
	}
}

func listAvailableSkills(workDir string) []string {
	var skills []string
	seen := make(map[string]bool)

	dirs := []string{
		filepath.Join(workDir, ".dcode", "skills"),
		filepath.Join(workDir, ".opencode", "skills"),
	}

	home, _ := os.UserHomeDir()
	if home != "" {
		dirs = append(dirs, filepath.Join(home, ".config", "dcode", "skills"))
	}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), ".md")
			if !seen[name] {
				seen[name] = true
				skills = append(skills, name)
			}
		}
	}

	return skills
}
