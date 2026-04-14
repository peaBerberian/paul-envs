package versions

import "github.com/peaberberian/paul-envs/internal/utils"

// Version of this application
// TODO: automatize
var Version = utils.Version{
	Major: 0,
	Minor: 6,
	Patch: 0,
}

// Version the Dockerfile and generated project files have as semver.
// It could be considered that project files have a dependency on the base
// Dockerfile. As such a new minor for base files is
// still compatible to older project files with the same major, but not
// vice-versa.
//
// # Changes
//   - 1.0.0: Base version
//   - 1.1.0: Added `INSTALL_OH_MY_POSH` env installing the `Oh My Posh` prompt
//   - 1.2.0: Added `INSTALL_CLAUDE_CODE` and `INSTALL_CODEX` envs
//   - 1.3.0: Added `INSTALL_OPEN_CODE`, `INSTALL_FIREFOX` envs
//   - 1.4.0: Added `userns_mode: keep-id` for Podman parity with Docker
//     Redirect in Dockerfile `.claude` and `.codex` to XDG_DATA_HOME for persistence
//   - 1.5.0: Set envs (XDG_* etc.) in global shellrc confs, so it's available in login shells
//   - 2.0.0: Replace per-project compose/env files with build.conf/run.conf
var DockerfileVersion = utils.Version{
	Major: 2,
	Minor: 0,
	Patch: 0,
}

// Format of the "project.lock" files: the lockfiles of the various projects.
var ProjectLockVersion = utils.Version{
	Major: 1,
	Minor: 0,
	Patch: 0,
}

// Format of the "project.buildinfo" files: Information on the last build performed for a project
var BuildInfoVersion = utils.Version{
	Major: 2,
	Minor: 0,
	Patch: 0,
}
