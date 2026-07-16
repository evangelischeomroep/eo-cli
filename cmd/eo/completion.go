package main

import (
	"fmt"
	"os"
)

func cmdCompletion(shell string) error {
	switch shell {
	case "zsh":
		fmt.Print(zshCompletion)
	case "bash":
		fmt.Print(bashCompletion)
	default:
		fmt.Fprintln(os.Stderr, bold("eo completion")+" — Output shell completion script")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, bold("USAGE"))
		fmt.Fprintln(os.Stderr, "  eo completion <shell>")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, bold("SHELLS"))
		fmt.Fprintln(os.Stderr, "  zsh   bash")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, bold("SETUP"))
		fmt.Fprintln(os.Stderr, dim("  # zsh — add to ~/.zshrc:"))
		fmt.Fprintln(os.Stderr, `  source <(eo completion zsh)`)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, dim("  # bash — add to ~/.bashrc:"))
		fmt.Fprintln(os.Stderr, `  source <(eo completion bash)`)
		if shell != "" {
			return fmt.Errorf("unknown shell %q — supported: zsh, bash", shell)
		}
	}
	return nil
}

const zshCompletion = `_eo() {
  local state

  _arguments \
    '1: :->command' \
    '*: :->args'

  case $state in
    command)
      local -a commands
      commands=(
        'deploy:Deploy Function Apps to test or prod'
        'pim:Activate the Contributor role for 8h'
        'whoami:Show the current Azure user and subscription'
        'version:Print the current version'
        'completion:Output shell completion script'
        'help:Show help for a command'
      )
      _describe 'command' commands
      ;;
    args)
      case $words[2] in
        deploy)
          if (( CURRENT == 3 )); then
            local -a envs
            envs=('test:Deploy to test' 'prod:Deploy to production')
            _describe 'environment' envs
          else
            _arguments '--all[Deploy all apps without selection]' '-a[Deploy all apps without selection]'
          fi
          ;;
        pim)
          local -a pim_cmds
          pim_cmds=('approve:List and approve pending PIM requests' 'status:Show if your Contributor role is active')
          _describe 'pim command' pim_cmds
          ;;
        completion)
          local -a shells
          shells=('zsh' 'bash')
          _describe 'shell' shells
          ;;
      esac
      ;;
  esac
}

compdef _eo eo
`

const bashCompletion = `_eo_completion() {
  local cur prev
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"

  case "${prev}" in
    eo)
      COMPREPLY=($(compgen -W "deploy pim whoami version completion help" -- "${cur}"))
      ;;
    deploy)
      COMPREPLY=($(compgen -W "test prod" -- "${cur}"))
      ;;
    test|prod)
      COMPREPLY=($(compgen -W "--all -a" -- "${cur}"))
      ;;
    pim)
      COMPREPLY=($(compgen -W "approve status" -- "${cur}"))
      ;;
    completion)
      COMPREPLY=($(compgen -W "zsh bash" -- "${cur}"))
      ;;
  esac
}

complete -F _eo_completion eo
`
