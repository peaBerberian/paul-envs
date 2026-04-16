package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/peaberberian/paul-envs/internal/files"
	"golang.org/x/term"
)

// Implements `ContainerEngine` for Docker.
type DockerEngine struct{}

func newDocker(ctx context.Context) (*DockerEngine, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, fmt.Errorf("docker command not found: %w", err)
	}
	return &DockerEngine{}, nil
}

func (c *DockerEngine) BuildImage(ctx context.Context, project files.ProjectEntry, options BuildOptions) error {
	buildCfg, err := loadBuildConfig(project)
	if err != nil {
		return err
	}

	cmdArgs := dockerBuildArgs(project, buildCfg.Args, options)
	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return fmt.Errorf("build failed: %w", err)
	}
	return nil
}

func dockerBuildArgs(project files.ProjectEntry, buildArgs map[string]string, options BuildOptions) []string {
	cmdArgs := []string{
		"build",
		"--file", projectDockerfilePath(project),
		"--tag", projectImageName(project.ProjectName),
	}
	if options.NoCache {
		cmdArgs = append(cmdArgs, "--no-cache")
	}

	keys := make([]string, 0, len(buildArgs))
	for key := range buildArgs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		cmdArgs = append(cmdArgs, "--build-arg", fmt.Sprintf("%s=%s", key, buildArgs[key]))
	}
	cmdArgs = append(cmdArgs, projectBaseDataDir(project))
	return cmdArgs
}

func (c *DockerEngine) RunContainer(ctx context.Context, project files.ProjectEntry, args []string) error {
	buildCfg, err := loadBuildConfig(project)
	if err != nil {
		return err
	}
	runtimeCfg, err := loadRuntimeConfig(project)
	if err != nil {
		return err
	}

	username := buildCfg.Args["USERNAME"]
	projectMount := projectMountTarget(username, project.ProjectName)
	workDir := runtimeCfg.WorkDir
	if workDir == "" {
		workDir = projectMount
	}

	if err := c.ensureVolumesExist(ctx, "paulenv-shared-cache", projectLocalVolumeName(project.ProjectName)); err != nil {
		return err
	}

	cmdArgs := []string{
		"run",
		"--rm",
		"--init",
		"--name", projectContainerName(project.ProjectName),
		"--workdir", workDir,
		"--volume", runtimeCfg.ProjectPath + ":" + projectMount,
		"--volume", "paulenv-shared-cache:/home/" + username + "/.container-cache",
		"--volume", projectLocalVolumeName(project.ProjectName) + ":/home/" + username + "/.container-local",
	}
	if runtimeCfg.DotfilesPath != "" {
		dotfilesPath, err := resolveRuntimePath(project.RuntimeConfigPath, runtimeCfg.DotfilesPath)
		if err != nil {
			return fmt.Errorf("resolve DOTFILES_PATH: %w", err)
		}
		cmdArgs = append(cmdArgs, "--volume", dotfilesPath+":/paul-env/dotfiles:ro")
	}
	if runtimeCfg.GitName != "" {
		cmdArgs = append(cmdArgs, "--env", "GIT_AUTHOR_NAME="+runtimeCfg.GitName)
	}
	if runtimeCfg.GitEmail != "" {
		cmdArgs = append(cmdArgs, "--env", "GIT_AUTHOR_EMAIL="+runtimeCfg.GitEmail)
	}

	for _, volume := range runtimeCfg.Volumes {
		cmdArgs = append(cmdArgs, "--volume", volume)
	}
	for _, port := range runtimeCfg.Ports {
		cmdArgs = append(cmdArgs, "--publish", port)
	}

	if term.IsTerminal(int(os.Stdin.Fd())) {
		cmdArgs = append(cmdArgs, "--tty", "--interactive")
	}

	cmdArgs = append(cmdArgs, projectImageName(project.ProjectName))
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return fmt.Errorf("run failed: %w", err)
	}
	return nil
}

func (c *DockerEngine) JoinContainer(ctx context.Context, containerInfo ContainerInfo, args []string) error {
	cmdArgs := []string{"exec"}
	if term.IsTerminal(int(os.Stdin.Fd())) {
		cmdArgs = append(cmdArgs, "-it")
	}
	cmdArgs = append(cmdArgs, containerInfo.ContainerId, "/usr/local/bin/entrypoint.sh")
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
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

func (c *DockerEngine) HasBeenBuilt(ctx context.Context, projectName string) (bool, error) {
	imageName := projectImageName(projectName)
	cmd := exec.CommandContext(ctx, "docker", "image", "inspect", imageName)
	err := cmd.Run()

	if err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return false, pErr
		}
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (c *DockerEngine) Info(ctx context.Context) (EngineInfo, error) {
	cmd := exec.CommandContext(ctx, "docker", "--version")
	output, err := cmd.Output()
	if err != nil {
		return EngineInfo{}, fmt.Errorf("failed to obtain docker version: %w", err)
	}
	parsed := strings.TrimSpace(string(output))
	re := regexp.MustCompile(`Docker version ([0-9]+\.[0-9]+\.[0-9]+)`)
	matches := re.FindStringSubmatch(parsed)
	if len(matches) > 1 {
		return EngineInfo{Version: matches[1], Name: "docker"}, nil
	}
	return EngineInfo{}, fmt.Errorf("failed to obtain docker version, unknown version format: %s", parsed)
}

func (c *DockerEngine) CreateVolume(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "docker", "volume", "create", name)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return fmt.Errorf("failed to create volume %s: %w", name, err)
	}
	return nil
}

func (c *DockerEngine) GetImageInfo(ctx context.Context, projectName string) (*ImageInfo, error) {
	imageName := projectImageName(projectName)
	info := &ImageInfo{ImageName: imageName, ProjectName: &projectName}

	cmd := exec.CommandContext(ctx, "docker", "image", "inspect", imageName, "--format", "{{.Created}}")
	output, err := cmd.Output()
	if err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return nil, pErr
		} else if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return info, nil
		}
		return nil, err
	}
	if buildTime, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(output))); err == nil {
		info.BuiltAt = &buildTime
	}
	return info, nil
}

func (c *DockerEngine) ListContainers(ctx context.Context) ([]ContainerInfo, error) {
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", "name=paulenv-", "--format", "{{.ID}}\t{{.Image}}\t{{.Names}}")
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
			projectName = projectNameFromContainerName(*name)
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

func (c *DockerEngine) RemoveContainer(ctx context.Context, container ContainerInfo) error {
	cmd := exec.CommandContext(ctx, "docker", "rm", "-f", container.ContainerId)
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return err
	}
	return nil
}

func (c *DockerEngine) checkPermissions(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "ps")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		if strings.Contains(stderrStr, "permission denied") ||
			strings.Contains(stderrStr, "access denied") ||
			strings.Contains(stderrStr, "dial unix") && strings.Contains(stderrStr, "connect: permission denied") {
			return errors.New("permission denied. Please run with elevated privileges")
		}
		return fmt.Errorf("failed to connect to Docker: %w\n%s", err, stderrStr)
	}
	return nil
}

func (c *DockerEngine) ListVolumes(ctx context.Context) ([]VolumeInfo, error) {
	cmd := exec.CommandContext(ctx, "docker", "volume", "ls", "--filter", "name=paulenv-", "--format", "{{.Name}}")
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

func (c *DockerEngine) RemoveVolume(ctx context.Context, volume VolumeInfo) error {
	cmd := exec.CommandContext(ctx, "docker", "volume", "rm", volume.VolumeName)
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return fmt.Errorf("failed to remove volume %s: %w", volume.VolumeName, err)
	}
	return nil
}

func (c *DockerEngine) ListNetworks(ctx context.Context) ([]NetworkInfo, error) {
	cmd := exec.CommandContext(ctx, "docker", "network", "ls", "--filter", "name=paulenv-", "--format", "{{.ID}}\t{{.Name}}")
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
		projectName := projectNameFromContainerName(parts[1])
		result = append(result, NetworkInfo{
			NetworkId:   parts[0],
			NetworkName: parts[1],
			ProjectName: projectName,
		})
	}
	return result, nil
}

func (c *DockerEngine) RemoveNetwork(ctx context.Context, network NetworkInfo) error {
	cmd := exec.CommandContext(ctx, "docker", "network", "rm", network.NetworkId)
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return fmt.Errorf("failed to remove network %s: %w", network.NetworkName, err)
	}
	return nil
}

func (c *DockerEngine) PruneBuildCache(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "builder", "prune", "-f", "--filter", "label=paulenv=true")
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return fmt.Errorf("failed to prune build cache: %w", err)
	}
	return nil
}

func (c *DockerEngine) SupportsBuildCachePrune() bool {
	return true
}

func (c *DockerEngine) ListImages(ctx context.Context) ([]ImageInfo, error) {
	cmd := exec.CommandContext(ctx, "docker", "images", "--filter", "reference=paulenv:*", "--format", "{{.Repository}}:{{.Tag}}\t{{.CreatedAt}}")
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

func (c *DockerEngine) RemoveImage(ctx context.Context, image ImageInfo) error {
	cmd := exec.CommandContext(ctx, "docker", "rmi", "-f", image.ImageName)
	if err := cmd.Run(); err != nil {
		if pErr := c.checkPermissions(ctx); pErr != nil {
			return pErr
		}
		return fmt.Errorf("failed to remove image %s: %w", image.ImageName, err)
	}
	return nil
}

func (c *DockerEngine) ensureVolumesExist(ctx context.Context, names ...string) error {
	volumes, err := c.ListVolumes(ctx)
	if err != nil {
		return fmt.Errorf("failed to list created volumes: %w", err)
	}

	existing := make([]string, 0, len(volumes))
	for _, volume := range volumes {
		existing = append(existing, volume.VolumeName)
	}

	for _, name := range names {
		if slices.Contains(existing, name) {
			continue
		}
		if err := c.CreateVolume(ctx, name); err != nil {
			return err
		}
	}
	return nil
}

func projectNameFromContainerName(containerName string) *string {
	if !strings.HasPrefix(containerName, "paulenv-") {
		return nil
	}
	projectName := strings.TrimPrefix(containerName, "paulenv-")
	if projectName == "" {
		return nil
	}
	return &projectName
}
