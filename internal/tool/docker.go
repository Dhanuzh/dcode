package tool

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// DockerTool provides Docker container operations
func DockerTool() *ToolDef {
	return &ToolDef{
		Name:        "Docker",
		Description: "Manage Docker containers, images, volumes, and networks.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"operation": map[string]interface{}{
					"type":        "string",
					"description": "Docker operation to perform",
					"enum": []string{
						"ps",      // List containers
						"images",  // List images
						"build",   // Build image
						"run",     // Run container
						"exec",    // Execute in container
						"logs",    // View logs
						"stop",    // Stop container
						"start",   // Start container
						"restart", // Restart container
						"rm",      // Remove container
						"rmi",     // Remove image
						"pull",    // Pull image
						"push",    // Push image
						"inspect", // Inspect container/image
						"stats",   // View stats
						"compose", // Docker compose
						"network", // Manage networks
						"volume",  // Manage volumes
					},
				},
				"args": map[string]interface{}{
					"type":        "array",
					"description": "Additional arguments for the Docker command",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"container": map[string]interface{}{
					"type":        "string",
					"description": "Container name or ID",
				},
				"image": map[string]interface{}{
					"type":        "string",
					"description": "Image name or ID",
				},
				"command": map[string]interface{}{
					"type":        "string",
					"description": "Command to run in container (for run/exec)",
				},
			},
			"required": []string{"operation"},
		},
		Execute: executeDocker,
	}
}

func executeDocker(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
	operation, ok := input["operation"].(string)
	if !ok {
		return &ToolResult{
			Output:  "operation parameter is required",
			IsError: true,
		}, nil
	}

	// Check if docker is available
	if !isCommandAvailable("docker") {
		return &ToolResult{
			Output:  "Docker is not installed or not in PATH",
			IsError: true,
		}, nil
	}

	// Build docker command
	var cmdArgs []string

	switch operation {
	case "ps":
		cmdArgs = []string{"ps"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		} else {
			cmdArgs = append(cmdArgs, "-a") // Show all containers by default
		}

	case "images":
		cmdArgs = []string{"images"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		}

	case "build":
		cmdArgs = []string{"build"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		} else {
			cmdArgs = append(cmdArgs, ".")
		}

	case "run":
		cmdArgs = []string{"run"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		}
		if image, ok := input["image"].(string); ok {
			cmdArgs = append(cmdArgs, image)
		}
		if command, ok := input["command"].(string); ok {
			cmdArgs = append(cmdArgs, strings.Fields(command)...)
		}

	case "exec":
		cmdArgs = []string{"exec"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		}
		container, ok := input["container"].(string)
		if !ok {
			return &ToolResult{
				Output:  "container parameter is required for exec",
				IsError: true,
			}, nil
		}
		cmdArgs = append(cmdArgs, container)
		if command, ok := input["command"].(string); ok {
			cmdArgs = append(cmdArgs, strings.Fields(command)...)
		}

	case "logs":
		cmdArgs = []string{"logs"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		}
		container, ok := input["container"].(string)
		if !ok {
			return &ToolResult{
				Output:  "container parameter is required for logs",
				IsError: true,
			}, nil
		}
		cmdArgs = append(cmdArgs, container)

	case "stop", "start", "restart":
		cmdArgs = []string{operation}
		container, ok := input["container"].(string)
		if !ok {
			return &ToolResult{
				Output:  fmt.Sprintf("container parameter is required for %s", operation),
				IsError: true,
			}, nil
		}
		cmdArgs = append(cmdArgs, container)

	case "rm":
		cmdArgs = []string{"rm"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		}
		container, ok := input["container"].(string)
		if !ok {
			return &ToolResult{
				Output:  "container parameter is required for rm",
				IsError: true,
			}, nil
		}
		cmdArgs = append(cmdArgs, container)

	case "rmi":
		cmdArgs = []string{"rmi"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		}
		image, ok := input["image"].(string)
		if !ok {
			return &ToolResult{
				Output:  "image parameter is required for rmi",
				IsError: true,
			}, nil
		}
		cmdArgs = append(cmdArgs, image)

	case "pull", "push":
		cmdArgs = []string{operation}
		image, ok := input["image"].(string)
		if !ok {
			return &ToolResult{
				Output:  fmt.Sprintf("image parameter is required for %s", operation),
				IsError: true,
			}, nil
		}
		cmdArgs = append(cmdArgs, image)

	case "inspect":
		cmdArgs = []string{"inspect"}
		target := ""
		if container, ok := input["container"].(string); ok {
			target = container
		} else if image, ok := input["image"].(string); ok {
			target = image
		} else {
			return &ToolResult{
				Output:  "container or image parameter is required for inspect",
				IsError: true,
			}, nil
		}
		cmdArgs = append(cmdArgs, target)

	case "stats":
		cmdArgs = []string{"stats"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		} else {
			cmdArgs = append(cmdArgs, "--no-stream")
		}
		if container, ok := input["container"].(string); ok {
			cmdArgs = append(cmdArgs, container)
		}

	case "compose":
		cmdArgs = []string{"compose"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		}

	case "network":
		cmdArgs = []string{"network"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		} else {
			cmdArgs = append(cmdArgs, "ls")
		}

	case "volume":
		cmdArgs = []string{"volume"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		} else {
			cmdArgs = append(cmdArgs, "ls")
		}

	default:
		return &ToolResult{
			Output:  fmt.Sprintf("unknown Docker operation: %s", operation),
			IsError: true,
		}, nil
	}

	// Execute docker command
	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	cmd.Dir = tc.WorkDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &ToolResult{
			Output:  fmt.Sprintf("Docker error: %s\nOutput: %s", err, string(output)),
			IsError: true,
		}, nil
	}

	return &ToolResult{
		Output:  string(output),
		IsError: false,
	}, nil
}

// Helper functions for Docker

// IsDockerRunning checks if Docker daemon is running
func IsDockerRunning() bool {
	cmd := exec.Command("docker", "info")
	err := cmd.Run()
	return err == nil
}

// GetDockerVersion returns the Docker version
func GetDockerVersion() (string, error) {
	cmd := exec.Command("docker", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
