package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/peaberberian/paul-envs/internal/files"
)

// Implements `ContainerEngine` for podman compose
type PodmanEngine struct{}

func newPodman(ctx context.Context) (*PodmanEngine, error) {
	// Check if podman compose is available (built-in since Podman 4.0)
	cmd := exec.CommandContext(ctx, "podman", "compose", "version")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("podman compose command not found: %w", err)
	}
	return &PodmanEngine{}, nil
}

func (c *PodmanEngine) BuildImage(ctx context.Context, project files.ProjectEntry, relativeDotfilesDir string) error {
	cmd := exec.CommandContext(ctx, "podman", "compose", "-f", project.ComposeFilePath, "--env-file", project.EnvFilePath, "build")
	envVars := append(os.Environ(),
		"COMPOSE_PROJECT_NAME=paulenv-"+project.ProjectName,
		"DOTFILES_DIR="+relativeDotfilesDir,
	)
	cmd.Env = envVars
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return fmt.Errorf("Build failed: %w", err)
	}
	return nil
}

func (c *PodmanEngine) RunContainer(ctx context.Context, project files.ProjectEntry, args []string) error {
	cmdArgs := []string{"compose", "-f", project.ComposeFilePath, "--env-file", project.EnvFilePath, "run", "--rm", "paulenv"}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.CommandContext(ctx, "podman", cmdArgs...)
	cmd.Env = append(os.Environ(), "COMPOSE_PROJECT_NAME=paulenv-"+project.ProjectName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return fmt.Errorf("Run failed: %w", err)
	}
	return nil
}

func (c *PodmanEngine) JoinContainer(ctx context.Context, containerInfo ContainerInfo, args []string) error {
	cmdArgs := []string{"exec", "-it", containerInfo.ContainerId, "/usr/local/bin/entrypoint.sh"}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.CommandContext(ctx, "podman", cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return fmt.Errorf("join exited: %w", err)
	}
	return nil
}

func (c *PodmanEngine) HasBeenBuilt(ctx context.Context, projectName string) (bool, error) {
	imageName := fmt.Sprintf("paulenv:%s", projectName)
	cmd := exec.CommandContext(ctx, "podman", "image", "inspect", imageName)
	err := cmd.Run()

	if err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return false, pErr
		}
		// Check if it's a "not found" error vs other error
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 125 || exitErr.ExitCode() == 1 {
				// Image doesn't exist (Podman typically uses 125 for not found)
				return false, nil
			}
		}
		// Other error
		return false, err
	}

	return true, nil
}

func (c *PodmanEngine) Info(ctx context.Context) (EngineInfo, error) {
	cmd := exec.CommandContext(ctx, "podman", "--version")
	output, err := cmd.Output()
	if err != nil {
		return EngineInfo{}, fmt.Errorf("failed to obtain podman version: %w", err)
	}
	parsed := strings.TrimSpace(fmt.Sprintf("%s", output))
	// Podman version output: "podman version 4.3.1"
	re := regexp.MustCompile(`podman version ([0-9]+\.[0-9]+\.[0-9]+)`)
	matches := re.FindStringSubmatch(string(parsed))
	if len(matches) > 1 {
		version := matches[1]
		return EngineInfo{Version: version, Name: "podman"}, nil
	}
	return EngineInfo{}, fmt.Errorf("failed to obtain podman version, unknown version format: %s", parsed)
}

func (c *PodmanEngine) CreateVolume(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "podman", "volume", "create", name)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return fmt.Errorf("Failed to create shared volume: %w.", err)
	}
	return nil
}

func (c *PodmanEngine) GetImageInfo(ctx context.Context, projectName string) (*ImageInfo, error) {
	imageName := fmt.Sprintf("paulenv:%s", projectName)
	info := &ImageInfo{ImageName: imageName, ProjectName: &projectName}

	// Check if image exists and get build time
	cmd := exec.CommandContext(ctx, "podman", "image", "inspect", imageName, "--format", "{{.Created}}")
	output, err := cmd.Output()
	if err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return nil, pErr
		} else if exitErr, ok := err.(*exec.ExitError); ok && (exitErr.ExitCode() == 125 || exitErr.ExitCode() == 1) {
			return info, nil
		}
		return nil, err
	}
	if buildTime, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(output))); err == nil {
		info.BuiltAt = &buildTime
	}
	return info, nil
}

func (c *PodmanEngine) ListContainers(ctx context.Context) ([]ContainerInfo, error) {
	cmd := exec.CommandContext(ctx, "podman", "ps", "-a", "--filter", "name=paulenv-", "--format", "{{.ID}}\t{{.Image}}\t{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return []ContainerInfo{}, pErr
		}
		return []ContainerInfo{}, fmt.Errorf("failed to list containers: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	result := make([]ContainerInfo, 0, len(lines))
	for _, s := range lines {
		if s != "" {
			parts := strings.SplitN(s, "\t", 3)
			id := parts[0]
			var image *string
			var name *string
			var projectName *string

			if len(parts) > 1 {
				image = &parts[1]
			}
			if len(parts) > 2 {
				name = &parts[2]
			}

			if image != nil && strings.HasPrefix(*image, "paulenv:") && len(*image) > len("paulenv:") {
				sliced := (*image)[len("paulenv:"):]
				projectName = &sliced
			}

			result = append(result, ContainerInfo{
				ProjectName:   projectName,
				ContainerName: name,
				ContainerId:   id,
				ImageName:     image,
			})
		}
	}
	return result, nil
}

func (c *PodmanEngine) RemoveContainer(ctx context.Context, container ContainerInfo) error {
	cmd := exec.CommandContext(ctx, "podman", "rm", "-f", container.ContainerId)
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return err
	}
	return nil
}

func (c *PodmanEngine) checkPermissions(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "podman", "ps")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		// Podman-specific permission errors
		if strings.Contains(stderrStr, "permission denied") ||
			strings.Contains(stderrStr, "access denied") ||
			strings.Contains(stderrStr, "cannot connect to Podman") {
			return errors.New("permission denied. Please check Podman socket permissions")
		}
		return fmt.Errorf("failed to connect to Podman: %w\n%s", err, stderrStr)
	}
	return nil
}

// List volumes currently known by this container engine
func (c *PodmanEngine) ListVolumes(ctx context.Context) ([]VolumeInfo, error) {
	cmd := exec.CommandContext(ctx, "podman", "volume", "ls", "--filter", "name=paulenv-", "--format", "{{.Name}}")
	output, err := cmd.Output()
	if err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return []VolumeInfo{}, pErr
		}
		return []VolumeInfo{}, fmt.Errorf("failed to list volumes: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	result := make([]VolumeInfo, 0, len(lines))
	for _, volumeName := range lines {
		if volumeName != "" {
			result = append(result, VolumeInfo{
				VolumeId:   volumeName,
				VolumeName: volumeName,
			})
		}
	}
	return result, nil
}

func (c *PodmanEngine) RemoveVolume(ctx context.Context, volume VolumeInfo) error {
	cmd := exec.CommandContext(ctx, "podman", "volume", "rm", volume.VolumeName)
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return fmt.Errorf("failed to remove volume %s: %w", volume.VolumeName, err)
	}
	return nil
}

// List networks currently known by this container engine
func (c *PodmanEngine) ListNetworks(ctx context.Context) ([]NetworkInfo, error) {
	cmd := exec.CommandContext(ctx, "podman", "network", "ls", "--filter", "name=paulenv-", "--format", "{{.ID}}\t{{.Name}}")
	output, err := cmd.Output()
	if err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return []NetworkInfo{}, pErr
		}
		return []NetworkInfo{}, fmt.Errorf("failed to list networks: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	result := make([]NetworkInfo, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) >= 2 {
				networkId := parts[0]
				networkName := parts[1]
				var projectName *string

				// Extract project name from network name if it follows the pattern "paulenv-{project}_default"
				if strings.HasPrefix(networkName, "paulenv-") {
					// Remove "paulenv-" prefix
					withoutPrefix := networkName[len("paulenv-"):]
					// Remove "_default" suffix if present
					if strings.HasSuffix(withoutPrefix, "_default") {
						sliced := withoutPrefix[:len(withoutPrefix)-len("_default")]
						projectName = &sliced
					} else {
						projectName = &withoutPrefix
					}
				}

				result = append(result, NetworkInfo{
					NetworkId:   networkId,
					NetworkName: networkName,
					ProjectName: projectName,
				})
			}
		}
	}
	return result, nil
}

// Remove network listed from this container engine
func (c *PodmanEngine) RemoveNetwork(ctx context.Context, network NetworkInfo) error {
	cmd := exec.CommandContext(ctx, "podman", "network", "rm", network.NetworkId)
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return fmt.Errorf("failed to remove network %s: %w", network.NetworkName, err)
	}
	return nil
}

// Remove the `ContainerEngine`'s build cache from metadata linked to this
// executable
func (c *PodmanEngine) PruneBuildCache(ctx context.Context) error {
	// Prune build cache for paulenv images specifically
	// Note: Podman's builder prune command syntax
	cmd := exec.CommandContext(ctx, "podman", "image", "prune", "-f", "--filter", "label=paulenv=true")
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return fmt.Errorf("failed to prune build cache: %w", err)
	}
	return nil
}

// List images currently known by this container engine
func (c *PodmanEngine) ListImages(ctx context.Context) ([]ImageInfo, error) {
	cmd := exec.CommandContext(ctx, "podman", "images", "--filter", "reference=paulenv:*", "--format", "{{.Repository}}:{{.Tag}}\t{{.CreatedAt}}")
	output, err := cmd.Output()
	if err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return []ImageInfo{}, pErr
		}
		return []ImageInfo{}, fmt.Errorf("failed to list images: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	result := make([]ImageInfo, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			parts := strings.SplitN(line, "\t", 2)
			imageName := parts[0]
			var projectName *string
			var builtAt *time.Time

			// Extract project name from image name if it follows the pattern "paulenv:{project}"
			if strings.HasPrefix(imageName, "paulenv:") && len(imageName) > len("paulenv:") {
				sliced := imageName[len("paulenv:"):]
				projectName = &sliced
			}

			// Parse build time if available
			if len(parts) > 1 {
				timeStr := strings.TrimSpace(parts[1])
				// Podman's CreatedAt format can vary, try common formats
				formats := []string{
					"2006-01-02 15:04:05 -0700 MST",
					time.RFC3339,
					time.RFC3339Nano,
				}
				for _, format := range formats {
					if parsedTime, err := time.Parse(format, timeStr); err == nil {
						builtAt = &parsedTime
						break
					}
				}
			}

			result = append(result, ImageInfo{
				ImageName:   imageName,
				ProjectName: projectName,
				BuiltAt:     builtAt,
			})
		}
	}
	return result, nil
}

// Remove image from this container engine
func (c *PodmanEngine) RemoveImage(ctx context.Context, image ImageInfo) error {
	cmd := exec.CommandContext(ctx, "podman", "rmi", "-f", image.ImageName)
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return fmt.Errorf("failed to remove image %s: %w", image.ImageName, err)
	}
	return nil
}
