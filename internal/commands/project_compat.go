package commands

import (
	"errors"
	"fmt"
	"os"
	"strings"

	versions "github.com/peaberberian/paul-envs/internal"
	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/files"
)

func ensureProjectCompatible(projectName string, filestore *files.FileStore, console *console.Console) error {
	status, err := filestore.ValidateProjectLock(projectName)
	if !status.IsValid() {
		reason := status.String()
		if err != nil {
			reason = err.Error()
		}
		return offerProjectReinitialize(projectName, filestore, console,
			fmt.Sprintf("This project was created with an incompatible format (%s).", reason))
	}

	_, err = filestore.ReadBuildInfo(projectName)
	if err == nil {
		return nil
	}
	if os.IsNotExist(err) || strings.Contains(err.Error(), "could not open 'project.buildinfo'") {
		return nil
	}
	if strings.Contains(err.Error(), "unknown 'project.buildinfo' version") ||
		strings.Contains(err.Error(), "invalid 'project.buildinfo' version") {
		return offerProjectReinitialize(projectName, filestore, console,
			fmt.Sprintf("This project uses an incompatible build metadata format. paul-envs now expects project.buildinfo version %s.", versions.BuildInfoVersion.ToString()))
	}
	return nil
}

func offerProjectReinitialize(projectName string, filestore *files.FileStore, console *console.Console, reason string) error {
	console.Warn("%s", reason)
	console.Warn("Automatic migration is not supported for this project format.")
	choice, err := console.AskYesNo(
		fmt.Sprintf("Wipe project '%s' so you can recreate it with 'paul-envs create'?", projectName),
		false,
	)
	if err != nil {
		return err
	}
	if choice {
		if err := filestore.DeleteProjectDirectory(projectName); err != nil {
			return fmt.Errorf("failed to remove incompatible project directory: %w", err)
		}
		return fmt.Errorf("project '%s' was removed; recreate it with 'paul-envs create'", projectName)
	}
	return errors.New("project format is incompatible; re-create it with 'paul-envs create'")
}
