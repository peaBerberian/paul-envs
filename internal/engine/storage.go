package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
)

const imageInfoFormat = `{"Created":{{json .Created}},"Size":{{json .Size}}}`

func parseImageInfo(output []byte, info *ImageInfo) error {
	var data struct {
		Created string
		Size    *int64
	}
	if err := json.Unmarshal(output, &data); err != nil {
		return fmt.Errorf("parse image information: %w", err)
	}
	info.BuiltAt = parseCreatedAt(data.Created)
	info.SizeBytes = nonNegativeSize(data.Size)
	return nil
}

func collectStorageUsage(ctx context.Context, binary string, projectNames []string, volumes map[string]*int64, volumeErr error, listContainers func(context.Context) ([]ContainerInfo, error)) (StorageUsage, error) {
	usage := StorageUsage{Projects: make(map[string]ProjectStorageUsage, len(projectNames))}
	if volumeErr == nil {
		usage.SharedCacheBytes = volumeBytes(volumes, sharedCacheVolumeName)
	}
	for _, project := range projectNames {
		var size ProjectStorageUsage
		if volumeErr == nil {
			size.PersistedBytes = volumeBytes(volumes, projectLocalVolumeName(project))
		}
		usage.Projects[project] = size
	}

	containers, err := listContainers(ctx)
	if err != nil {
		return usage, errors.Join(volumeErr, err)
	}
	// Inspect only containers belonging to the requested projects. All resource
	// naming and association stays in this package.
	var selected []ContainerInfo
	for _, container := range containers {
		if container.ProjectName != nil {
			if _, ok := usage.Projects[*container.ProjectName]; ok {
				selected = append(selected, container)
			}
		}
	}
	writable, containerErr := containerWritableBytes(ctx, binary, selected)
	for project, size := range usage.Projects {
		total := int64(0)
		size.TemporaryBytes = &total
		for _, container := range selected {
			if *container.ProjectName != project {
				continue
			}
			bytes := writable[container.ContainerId]
			if bytes == nil || *bytes > math.MaxInt64-total {
				size.TemporaryBytes = nil
				break
			}
			total += *bytes
		}
		usage.Projects[project] = size
	}
	return usage, errors.Join(volumeErr, containerErr)
}

// Called only after a successful volume query: absence means no stored data.
func volumeBytes(volumes map[string]*int64, name string) *int64 {
	if size, exists := volumes[name]; exists {
		return size
	}
	zero := int64(0)
	return &zero
}

func containerWritableBytes(ctx context.Context, binary string, containers []ContainerInfo) (map[string]*int64, error) {
	sizes := make(map[string]*int64, len(containers))
	if len(containers) == 0 {
		return sizes, nil
	}
	args := []string{"container", "inspect", "--size"}
	for _, container := range containers {
		args = append(args, container.ContainerId)
	}
	output, err := storageOutput(ctx, binary, args...)
	if err != nil {
		return nil, err
	}
	var inspected []struct {
		ID     string `json:"Id"`
		SizeRw *int64
	}
	if err := json.Unmarshal(output, &inspected); err != nil {
		return nil, fmt.Errorf("parse container sizes: %w", err)
	}
	for _, container := range containers {
		for _, item := range inspected {
			// ps can abbreviate IDs, whereas inspect returns full IDs.
			if container.ContainerId != "" && strings.HasPrefix(item.ID, container.ContainerId) {
				sizes[container.ContainerId] = nonNegativeSize(item.SizeRw)
				break
			}
		}
	}
	return sizes, nil
}

func storageOutput(ctx context.Context, binary string, args ...string) ([]byte, error) {
	output, err := exec.CommandContext(ctx, binary, args...).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("%s %s: %w: %s", binary, args[0], err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("%s %s: %w", binary, args[0], err)
	}
	return output, nil
}

// Both CLIs round volume sizes before returning them, including Docker's JSON.
func parseReportedSize(value string) *int64 {
	units := []struct {
		suffix string
		factor float64
	}{
		{"PiB", 1 << 50}, {"TiB", 1 << 40}, {"GiB", 1 << 30}, {"MiB", 1 << 20}, {"KiB", 1 << 10},
		{"PB", 1e15}, {"TB", 1e12}, {"GB", 1e9}, {"MB", 1e6}, {"kB", 1e3}, {"KB", 1e3}, {"B", 1},
	}
	for _, unit := range units {
		if !strings.HasSuffix(value, unit.suffix) {
			continue
		}
		number, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(value, unit.suffix)), 64)
		bytes := number * unit.factor
		if err != nil || math.IsNaN(bytes) || math.IsInf(bytes, 0) || bytes < 0 || bytes >= math.MaxInt64 {
			return nil
		}
		size := int64(math.Round(bytes))
		return &size
	}
	return nil
}

func nonNegativeSize(size *int64) *int64 {
	if size == nil || *size < 0 {
		return nil
	}
	return size
}
