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
	case "fish":
		fmt.Print(fishCompletion)
	default:
		fmt.Fprintln(os.Stderr, bold("eo completion")+" — Output shell completion script")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, bold("USAGE"))
		fmt.Fprintln(os.Stderr, "  eo completion <shell>")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, bold("SHELLS"))
		fmt.Fprintln(os.Stderr, "  zsh   bash   fish")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, bold("SETUP"))
		fmt.Fprintln(os.Stderr, dim("  # zsh — add to ~/.zshrc:"))
		fmt.Fprintln(os.Stderr, `  source <(eo completion zsh)`)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, dim("  # bash — add to ~/.bashrc:"))
		fmt.Fprintln(os.Stderr, `  source <(eo completion bash)`)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, dim("  # fish — save to the completions directory:"))
		fmt.Fprintln(os.Stderr, `  eo completion fish > ~/.config/fish/completions/eo.fish`)
		if shell != "" {
			return fmt.Errorf("unknown shell %q — supported: zsh, bash, fish", shell)
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
          shells=('zsh' 'bash' 'fish')
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
      COMPREPLY=($(compgen -W "zsh bash fish" -- "${cur}"))
      ;;
  esac
}

complete -F _eo_completion eo
`

const fishCompletion = `complete -c eo -f

complete -c eo -n '__fish_use_subcommand' -a deploy -d 'Deploy Function Apps to test or prod'
complete -c eo -n '__fish_use_subcommand' -a pim -d 'Activate the Contributor role for 8h'
complete -c eo -n '__fish_use_subcommand' -a whoami -d 'Show the current Azure user and subscription'
complete -c eo -n '__fish_use_subcommand' -a version -d 'Print the current version'
complete -c eo -n '__fish_use_subcommand' -a completion -d 'Output shell completion script'
complete -c eo -n '__fish_use_subcommand' -a help -d 'Show help for a command'

complete -c eo -n '__fish_seen_subcommand_from deploy; and not __fish_seen_subcommand_from test prod' -a test -d 'Deploy to test'
complete -c eo -n '__fish_seen_subcommand_from deploy; and not __fish_seen_subcommand_from test prod' -a prod -d 'Deploy to production'
complete -c eo -n '__fish_seen_subcommand_from test prod' -s a -l all -d 'Deploy all apps without selection'

complete -c eo -n '__fish_seen_subcommand_from pim; and not __fish_seen_subcommand_from approve status' -a approve -d 'List and approve pending PIM requests'
complete -c eo -n '__fish_seen_subcommand_from pim; and not __fish_seen_subcommand_from approve status' -a status -d 'Show if your Contributor role is active'

complete -c eo -n '__fish_seen_subcommand_from completion' -a 'zsh bash fish'
`
