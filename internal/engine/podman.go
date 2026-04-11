package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/peaberberian/paul-envs/internal/files"
	"golang.org/x/term"
)

// Implements `ContainerEngine` for Podman with a Compose-compatible frontend.
type PodmanEngine struct {
	composeCommand []string
}

func newPodman(ctx context.Context) (*PodmanEngine, error) {
	if _, err := exec.LookPath("podman"); err != nil {
		return nil, fmt.Errorf("podman command not found: %w", err)
	}

	// `podman compose` is only a wrapper around an external provider and may pick
	// Docker's compose plugin, which then requires a Podman API socket. For the
	// rootless workflow we want here, prefer `podman-compose` explicitly.
	if _, err := exec.LookPath("podman-compose"); err == nil {
		cmd := exec.CommandContext(ctx, "podman-compose", "version")
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("podman-compose command not usable: %w", err)
		}
		return &PodmanEngine{composeCommand: []string{"podman-compose"}}, nil
	}

	return nil, errors.New("podman-compose is required for Podman support; install it or use docker")
}

func (c *PodmanEngine) BuildImage(ctx context.Context, project files.ProjectEntry, relativeDotfilesDir string) error {
	cmd, cleanup, err := c.composeCommandContext(
		ctx,
		project,
		"build",
	)
	if err != nil {
		return err
	}
	defer cleanup()
	cmd.Env = append(cmd.Env, "DOTFILES_DIR="+relativeDotfilesDir)
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
	cmdArgs := []string{"run", "--rm", "paulenv"}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		cmdArgs = append([]string{"--podman-run-args=--tty=false"}, cmdArgs...)
	}
	cmdArgs = append(cmdArgs, args...)
	cmd, cleanup, err := c.composeCommandContext(ctx, project, cmdArgs...)
	if err != nil {
		return err
	}
	defer cleanup()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
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
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 125 || exitErr.ExitCode() == 1 {
				return false, nil
			}
		}
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
	parsed := strings.TrimSpace(string(output))
	re := regexp.MustCompile(`podman version ([0-9]+\.[0-9]+\.[0-9]+)`)
	matches := re.FindStringSubmatch(parsed)
	if len(matches) > 1 {
		return EngineInfo{Version: matches[1], Name: "podman"}, nil
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
	imageName := fmt.Sprintf("localhost/paulenv:%s", projectName)
	info := &ImageInfo{ImageName: imageName, ProjectName: &projectName}
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
	if buildTime := parseCreatedAt(string(output)); buildTime != nil {
		info.BuiltAt = buildTime
	}
	return info, nil
}

func (c *PodmanEngine) ListContainers(ctx context.Context) ([]ContainerInfo, error) {
	cmd := exec.CommandContext(ctx, "podman", "ps", "-a", "--format", "{{.ID}}\t{{.Image}}\t{{.Names}}")
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
		if s == "" {
			continue
		}
		parts := strings.SplitN(s, "\t", 3)
		id := parts[0]
		var image *string
		var name *string
		var projectName *string

		if len(parts) > 1 && parts[1] != "" {
			image = &parts[1]
		}
		if len(parts) > 2 && parts[2] != "" {
			name = &parts[2]
		}

		if image != nil {
			projectName = projectNameFromImage(*image)
		}
		if projectName == nil && name != nil {
			projectName = projectNameFromComposeResource(*name)
		}
		if projectName == nil {
			continue
		}

		result = append(result, ContainerInfo{
			ProjectName:   projectName,
			ContainerName: name,
			ContainerId:   id,
			ImageName:     image,
		})
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
		if strings.Contains(stderrStr, "permission denied") ||
			strings.Contains(stderrStr, "access denied") ||
			strings.Contains(stderrStr, "cannot connect to Podman") {
			return errors.New("permission denied. Please check Podman permissions")
		}
		return fmt.Errorf("failed to connect to Podman: %w\n%s", err, stderrStr)
	}
	return nil
}

func (c *PodmanEngine) ListVolumes(ctx context.Context) ([]VolumeInfo, error) {
	cmd := exec.CommandContext(ctx, "podman", "volume", "ls", "--format", "{{.Name}}")
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
		if volumeName == "" || !isPaulEnvVolume(volumeName) {
			continue
		}
		result = append(result, VolumeInfo{
			VolumeId:   volumeName,
			VolumeName: volumeName,
		})
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

func (c *PodmanEngine) ListNetworks(ctx context.Context) ([]NetworkInfo, error) {
	cmd := exec.CommandContext(ctx, "podman", "network", "ls", "--format", "{{.ID}}\t{{.Name}}")
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
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) < 2 {
			continue
		}
		projectName := projectNameFromComposeResource(parts[1])
		if projectName == nil {
			continue
		}
		result = append(result, NetworkInfo{
			NetworkId:   parts[0],
			NetworkName: parts[1],
			ProjectName: projectName,
		})
	}
	return result, nil
}

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

func (c *PodmanEngine) PruneBuildCache(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "podman", "image", "prune", "-f", "--filter", "label=paulenv=true")
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return fmt.Errorf("failed to prune build cache: %w", err)
	}
	return nil
}

func (c *PodmanEngine) ListImages(ctx context.Context) ([]ImageInfo, error) {
	cmd := exec.CommandContext(ctx, "podman", "images", "--format", "{{.Repository}}:{{.Tag}}\t{{.CreatedAt}}")
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
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		imageName := parts[0]
		projectName := projectNameFromImage(imageName)
		if projectName == nil {
			continue
		}
		var builtAt *time.Time
		if len(parts) > 1 {
			builtAt = parseCreatedAt(parts[1])
		}

		result = append(result, ImageInfo{
			ImageName:   imageName,
			ProjectName: projectName,
			BuiltAt:     builtAt,
		})
	}
	return result, nil
}

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

func (c *PodmanEngine) composeCommandContext(ctx context.Context, project files.ProjectEntry, args ...string) (*exec.Cmd, func(), error) {
	cmdArgs := append([]string{}, c.composeCommand[1:]...)
	cmdArgs = append(cmdArgs,
		"-f", project.ComposeFilePath,
		"--env-file", project.EnvFilePath,
	)

	// TODO: CI too weird as an env, got tired of doing things well
	var overrideFilePath string
	if os.Getenv("CI") == "true" || !supportsKeepID() {
		newCompose, err := c.createComposeUserNsOverride(project)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create Podman compose override: %w", err)
		}
		overrideFilePath = newCompose
	}

	if overrideFilePath != "" {
		cmdArgs = append(cmdArgs, "-f", overrideFilePath)
	}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.CommandContext(ctx, c.composeCommand[0], cmdArgs...)
	cmd.Env = append(os.Environ(), "COMPOSE_PROJECT_NAME=paulenv-"+project.ProjectName)
	cleanup := func() {
		if overrideFilePath != "" {
			_ = os.Remove(overrideFilePath)
		}
	}
	return cmd, cleanup, nil
}

func (c *PodmanEngine) createComposeUserNsOverride(project files.ProjectEntry) (string, error) {
	overrideDir := filepath.Dir(project.ComposeFilePath)
	tmpFile, err := os.CreateTemp(overrideDir, "podman-compose-override-*.yaml")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	overrideContent := "services:\n  paulenv:\n"
	overrideContent += "    userns_mode: \"host\"\n"
	overrideContent += "\nx-podman:\n  in_pod: false\n"
	if _, err = tmpFile.WriteString(overrideContent); err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", err
	}
	return tmpFile.Name(), nil
}

func projectNameFromImage(imageName string) *string {
	if strings.HasPrefix(imageName, "paulenv:") && len(imageName) > len("paulenv:") {
		projectName := imageName[len("paulenv:"):]
		return &projectName
	}
	return nil
}

// NOTE: hack for a resilience edge case
// TODO: better document
func projectNameFromComposeResource(resourceName string) *string {
	if !strings.HasPrefix(resourceName, "paulenv-") {
		return nil
	}
	projectName := strings.TrimPrefix(resourceName, "paulenv-")

	switch {
	case strings.HasSuffix(projectName, "_default"):
		projectName = strings.TrimSuffix(projectName, "_default")
	case strings.HasSuffix(projectName, "-local"):
		projectName = strings.TrimSuffix(projectName, "-local")
	case strings.Contains(projectName, "-paulenv-"):
		projectName, _, _ = strings.Cut(projectName, "-paulenv-")
	case strings.Contains(projectName, "_paulenv_"):
		projectName, _, _ = strings.Cut(projectName, "_paulenv_")
	}

	if projectName == "" {
		return nil
	}
	return &projectName
}

func isPaulEnvVolume(volumeName string) bool {
	return volumeName == "paulenv-shared-cache" ||
		(strings.HasPrefix(volumeName, "paulenv-") && strings.HasSuffix(volumeName, "-local"))
}

func parseCreatedAt(timeStr string) *time.Time {
	formats := []string{
		"2006-01-02 15:04:05 -0700 MST",
		time.RFC3339,
		time.RFC3339Nano,
	}
	timeStr = strings.TrimSpace(timeStr)
	for _, format := range formats {
		if parsedTime, err := time.Parse(format, timeStr); err == nil {
			return &parsedTime
		}
	}
	return nil
}

func supportsKeepID() bool {
	if runtime.GOOS != "linux" {
		return true
	}
	data, err := os.ReadFile("/proc/sys/user/max_user_namespaces")
	if err != nil {
		return true
	}
	return strings.TrimSpace(string(data)) != "0"
}
