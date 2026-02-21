package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (a App) cmdCompletion(g globalFlags, args []string) error {
	if len(args) == 0 {
		return newExitError(ExitInvalidUsage, "usage: gflight completion <bash|zsh|fish> | gflight completion path <bash|zsh|fish>")
	}
	if len(args) == 2 && strings.EqualFold(args[0], "path") {
		p, err := completionInstallPath(args[1])
		if err != nil {
			return err
		}
		fmt.Println(p)
		return nil
	}
	if len(args) != 1 {
		return newExitError(ExitInvalidUsage, "usage: gflight completion <bash|zsh|fish> | gflight completion path <bash|zsh|fish>")
	}
	switch strings.ToLower(args[0]) {
	case "bash":
		fmt.Print(bashCompletionScript())
		return nil
	case "zsh":
		fmt.Print(zshCompletionScript())
		return nil
	case "fish":
		fmt.Print(fishCompletionScript())
		return nil
	default:
		return newExitError(ExitInvalidUsage, "unsupported shell %q (use bash, zsh, or fish)", args[0])
	}
}

func completionInstallPath(shell string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", newExitError(ExitGenericFailure, "cannot resolve user home directory")
	}
	switch strings.ToLower(strings.TrimSpace(shell)) {
	case "zsh":
		return filepath.Join(home, ".zsh", "completions", "_gflight"), nil
	case "bash":
		return filepath.Join(home, ".local", "share", "bash-completion", "completions", "gflight"), nil
	case "fish":
		return filepath.Join(home, ".config", "fish", "completions", "gflight.fish"), nil
	default:
		return "", newExitError(ExitInvalidUsage, "unsupported shell %q (use bash, zsh, or fish)", shell)
	}
}

func bashCompletionScript() string {
	return `#!/usr/bin/env bash
_gflight_completions() {
  local cur prev words cword
  _init_completion -n : || return

  local commands="search watch notify auth config completion doctor help version"
  local watch_sub="create list enable disable delete run test"
  local auth_sub="login status"
  local config_sub="get set"

  if [[ ${cword} -eq 1 ]]; then
    COMPREPLY=( $(compgen -W "${commands}" -- "${cur}") )
    return
  fi

  case "${words[1]}" in
    watch) COMPREPLY=( $(compgen -W "${watch_sub}" -- "${cur}") ) ;;
    auth) COMPREPLY=( $(compgen -W "${auth_sub}" -- "${cur}") ) ;;
    config) COMPREPLY=( $(compgen -W "${config_sub}" -- "${cur}") ) ;;
    completion) COMPREPLY=( $(compgen -W "bash zsh fish" -- "${cur}") ) ;;
  esac
}
complete -F _gflight_completions gflight
`
}

func zshCompletionScript() string {
	return `#compdef gflight
_gflight() {
  local -a commands
  commands=(
    'search:One-shot flight search'
    'watch:Manage watches'
    'notify:Test notifications'
    'auth:Manage provider auth'
    'config:Read or write config'
    'completion:Generate shell completion'
    'doctor:Run preflight checks'
    'help:Show help'
    'version:Show version'
  )

  local -a watch_sub
  watch_sub=('create' 'list' 'enable' 'disable' 'delete' 'run' 'test')
  local -a auth_sub
  auth_sub=('login' 'status')
  local -a config_sub
  config_sub=('get' 'set')

  if (( CURRENT == 2 )); then
    _describe 'command' commands
    return
  fi

  case "$words[2]" in
    watch) _describe 'watch command' watch_sub ;;
    auth) _describe 'auth command' auth_sub ;;
    config) _describe 'config action' config_sub ;;
    completion) _values 'shell' bash zsh fish ;;
  esac
}
_gflight "$@"
`
}

func fishCompletionScript() string {
	return `complete -c gflight -f
complete -c gflight -n '__fish_use_subcommand' -a 'search' -d 'One-shot flight search'
complete -c gflight -n '__fish_use_subcommand' -a 'watch' -d 'Manage watches'
complete -c gflight -n '__fish_use_subcommand' -a 'notify' -d 'Test notifications'
complete -c gflight -n '__fish_use_subcommand' -a 'auth' -d 'Manage provider auth'
complete -c gflight -n '__fish_use_subcommand' -a 'config' -d 'Read or write config'
complete -c gflight -n '__fish_use_subcommand' -a 'completion' -d 'Generate shell completion'
complete -c gflight -n '__fish_use_subcommand' -a 'doctor' -d 'Run preflight checks'
complete -c gflight -n '__fish_use_subcommand' -a 'help' -d 'Show help'
complete -c gflight -n '__fish_use_subcommand' -a 'version' -d 'Show version'

complete -c gflight -n '__fish_seen_subcommand_from watch' -a 'create list enable disable delete run test'
complete -c gflight -n '__fish_seen_subcommand_from auth' -a 'login status'
complete -c gflight -n '__fish_seen_subcommand_from config' -a 'get set'
complete -c gflight -n '__fish_seen_subcommand_from completion' -a 'bash zsh fish'
`
}
