package commands

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	versions "github.com/peaberberian/paul-envs/internal"
	"github.com/peaberberian/paul-envs/internal/args"
	"github.com/peaberberian/paul-envs/internal/config"
	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/files"
	"github.com/peaberberian/paul-envs/internal/utils"
)

func Create(argsList []string, filestore *files.FileStore, console *console.Console) error {
	if len(argsList) > 0 && isHelpArg(argsList[0]) {
		args.WriteCreateUsage(console)
		return nil
	}

	cfg, err := args.ParseAndPrompt(argsList, console, filestore)
	if err != nil {
		return err
	}

	if err := generateProjectFiles(&cfg, filestore); err != nil {
		return err
	}
	if cfg.SeedDotfiles {
		if err := filestore.SeedProjectDotfiles(context.Background(), cfg.ProjectName); err != nil {
			return fmt.Errorf("failed to seed project dotfiles: %w", err)
		}
	}
	printNextSteps(&cfg, filestore, console)
	return nil
}

func generateProjectFiles(cfg *config.Config, filestore *files.FileStore) error {
	if filestore.DoesProjectExist(cfg.ProjectName) {
		return errors.New("project name already taken")
	}

	// TODO: Should those template definitions be moved to the `FileStore` code?
	// It could only take the Config as argument
	buildData := files.BuildTemplateData{
		Version:           versions.BuildConfigVersion.ToString(),
		HostUID:           utils.EscapeEnvValue(cfg.UID),
		HostGID:           utils.EscapeEnvValue(cfg.GID),
		Username:          utils.EscapeEnvValue(cfg.Username),
		Shell:             string(cfg.Shell),
		InstallNode:       utils.EscapeEnvValue(cfg.InstallNode),
		InstallRust:       utils.EscapeEnvValue(cfg.InstallRust),
		InstallPython:     utils.EscapeEnvValue(cfg.InstallPython),
		InstallGo:         utils.EscapeEnvValue(cfg.InstallGo),
		EnableWasm:        strconv.FormatBool(cfg.EnableWasm),
		EnableSSH:         strconv.FormatBool(cfg.EnableSsh),
		EnableSudo:        strconv.FormatBool(cfg.EnableSudo),
		Packages:          utils.EscapeEnvValue(strings.Join(cfg.Packages, " ")),
		InstallNeovim:     strconv.FormatBool(cfg.InstallNeovim),
		InstallStarship:   strconv.FormatBool(cfg.InstallStarship),
		InstallOhMyPosh:   strconv.FormatBool(cfg.InstallOhMyPosh),
		InstallAtuin:      strconv.FormatBool(cfg.InstallAtuin),
		InstallMise:       strconv.FormatBool(cfg.InstallMise),
		InstallZellij:     strconv.FormatBool(cfg.InstallZellij),
		InstallJujutsu:    strconv.FormatBool(cfg.InstallJujutsu),
		InstallDelta:      strconv.FormatBool(cfg.InstallDelta),
		InstallOpenCode:   strconv.FormatBool(cfg.InstallOpenCode),
		InstallClaudeCode: strconv.FormatBool(cfg.InstallClaudeCode),
		InstallCodex:      strconv.FormatBool(cfg.InstallCodex),
		InstallFirefox:    strconv.FormatBool(cfg.InstallFirefox),
	}

	runtimeData := files.RuntimeTemplateData{
		Version:         versions.RuntimeConfigVersion.ToString(),
		ProjectHostPath: utils.EscapeEnvValue(cfg.ProjectHostPath),
		DotfilesPath:    "dotfiles",
		GitName:         utils.EscapeEnvValue(cfg.GitName),
		GitEmail:        utils.EscapeEnvValue(cfg.GitEmail),
		Volumes:         cfg.Volumes,
		Ports:           runtimePorts(cfg.Ports),
	}

	err := filestore.CreateProjectFiles(cfg.ProjectName, buildData, runtimeData)
	if err != nil {
		return fmt.Errorf("failed to create project files: %w", err)
	}
	return nil
}

func runtimePorts(ports []uint16) []string {
	out := make([]string, 0, len(ports))
	for _, port := range ports {
		out = append(out, fmt.Sprintf("%d:%d", port, port))
	}
	return out
}

func isHelpArg(arg string) bool {
	return arg == "--help" || arg == "-h"
}

func printNextSteps(cfg *config.Config, filestore *files.FileStore, console *console.Console) {
	console.Success("Created project '%s'", cfg.ProjectName)
	console.WriteLn("")
	console.WriteLn("Next steps:")
	console.WriteLn("  1. Review/edit configuration:")
	// TODO: rely on just `GetProject` instead
	console.WriteLn("     - %s", filestore.GetProjectBuildConfigPath(cfg.ProjectName))
	console.WriteLn("     - %s", filestore.GetProjectRuntimeConfigPath(cfg.ProjectName))
	console.WriteLn("  2. Optionally add project-specific dotfiles:")
	console.WriteLn("     - %s", filestore.GetProjectDotfilesPath(cfg.ProjectName))
	console.WriteLn("     These are synced when the environment starts, so you can do this now or later.")
	console.WriteLn("  3. Build the environment:")
	console.WriteLn("     paul-envs build %s", cfg.ProjectName)
	console.WriteLn("  4. Run the environment:")
	console.WriteLn("     paul-envs run %s", cfg.ProjectName)
}
