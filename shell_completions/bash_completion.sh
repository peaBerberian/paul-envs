_paulenvs()
{
    local cur prev opts create_opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    # Main commands
    local commands="create list build run remove version interactive help clean"

    # Options for create command
    local create_flags="--help --name --uid --gid --username --shell --nodejs --rust --python --go --git-name --git-email --package --enable-ssh --enable-sudo --neovim --starship --oh-my-posh --atuin --zellij --jujutsu --delta --open-code --claude-code --codex --firefox --no-mise --port --volume"

    # Options for list command
    local list_flags="--help --names"
    local build_flags="--help --no-cache"
    local remove_flags="--help --no-prompt"
    local clean_flags="--help --no-prompt --engine"
    local engine_values="docker podman all"
    local run_flags="--help"
    local version_flags="--help"
    local interactive_flags="--help"

    # Get list of existing containers from paul-envs ls
    _get_containers() {
        paul-envs list --names 2>/dev/null
    }

    # First argument (command)
    if [[ $COMP_CWORD -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${commands}" -- ${cur}) )
        return 0
    fi

    local command="${COMP_WORDS[1]}"

    case "${command}" in
        create)
            case "${prev}" in
                --uid|--gid)
                    # Could suggest current UID/GID
                    COMPREPLY=( $(compgen -W "$(id -u) $(id -g)" -- ${cur}) )
                    return 0
                    ;;
                --username|--git-name|--git-email|--package|--nodejs|--rust|--python|--go|--port)
                    # Let user type freely
                    COMPREPLY=()
                    return 0
                    ;;
                --shell)
                    COMPREPLY=( $(compgen -W "bash zsh fish" -- ${cur}) )
                    return 0
                    ;;
                --volume)
                    # Complete file paths
                    COMPREPLY=( $(compgen -f -- ${cur}) )
                    return 0
                    ;;
                create)
                    # After 'create', expect project name (no completion)
                    COMPREPLY=()
                    return 0
                    ;;
                *)
                    # If previous was a project name, suggest path completion
                    # Otherwise suggest flags
                    if [[ $COMP_CWORD -eq 3 ]]; then
                        # Third argument: project path
                        COMPREPLY=( $(compgen -d -- ${cur}) )
                    else
                        # Suggest create flags
                        COMPREPLY=( $(compgen -W "${create_flags}" -- ${cur}) )
                    fi
                    return 0
                    ;;
            esac
            ;;
        list)
            # Suggest list flags
            COMPREPLY=( $(compgen -W "${list_flags}" -- ${cur}) )
            return 0
            ;;
        build)
            if [[ $COMP_CWORD -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "$(_get_containers) ${build_flags}" -- ${cur}) )
            elif [[ "${cur}" == --* ]]; then
                COMPREPLY=( $(compgen -W "${build_flags}" -- ${cur}) )
            fi
            return 0
            ;;
        run)
            if [[ $COMP_CWORD -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "$(_get_containers) ${run_flags}" -- ${cur}) )
            elif [[ "${cur}" == --* ]]; then
                COMPREPLY=( $(compgen -W "${run_flags}" -- ${cur}) )
            fi
            return 0
            ;;
        remove)
            if [[ $COMP_CWORD -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "$(_get_containers) ${remove_flags}" -- ${cur}) )
            elif [[ "${cur}" == --* ]]; then
                COMPREPLY=( $(compgen -W "${remove_flags}" -- ${cur}) )
            fi
            return 0
            ;;
        version)
            COMPREPLY=( $(compgen -W "${version_flags}" -- ${cur}) )
            return 0
            ;;
        interactive)
            COMPREPLY=( $(compgen -W "${interactive_flags}" -- ${cur}) )
            return 0
            ;;
        clean)
            if [[ "${prev}" == --engine ]]; then
                COMPREPLY=( $(compgen -W "${engine_values}" -- ${cur}) )
            else
                COMPREPLY=( $(compgen -W "${clean_flags}" -- ${cur}) )
            fi
            return 0
            ;;
        help)
            # No further completion
            return 0
            ;;
    esac
}

complete -F _paulenvs paul-envs
