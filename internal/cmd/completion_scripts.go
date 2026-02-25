package cmd

import "fmt"

func completionScript(shell string) (string, error) {
	switch shell {
	case "bash":
		return bashCompletionScript(), nil
	case "zsh":
		return zshCompletionScript(), nil
	case "fish":
		return fishCompletionScript(), nil
	case "powershell":
		return powerShellCompletionScript(), nil
	default:
		return "", fmt.Errorf("unsupported shell: %s", shell)
	}
}

func bashCompletionScript() string {
	return `#!/usr/bin/env bash

_wk_complete() {
  local IFS=$'\n'
  local completions
  completions=$(wk __complete --cword "$COMP_CWORD" -- "${COMP_WORDS[@]}")
  COMPREPLY=()
  if [[ -n "$completions" ]]; then
    COMPREPLY=( $completions )
  fi
}

complete -F _wk_complete wk
`
}

func zshCompletionScript() string {
	return `#compdef wk

autoload -Uz bashcompinit
bashcompinit
` + bashCompletionScript()
}

func fishCompletionScript() string {
	return `function __wk_complete
  set -l words (commandline -opc)
  set -l cur (commandline -ct)
  set -l cword (count $words)
  if test -n "$cur"
    set cword (math $cword - 1)
  end
  wk __complete --cword $cword -- $words
end

complete -c wk -f -a "(__wk_complete)"
`
}

func powerShellCompletionScript() string {
	return `Register-ArgumentCompleter -CommandName wk -ScriptBlock {
  param($commandName, $wordToComplete, $cursorPosition, $commandAst, $fakeBoundParameter)
  $elements = $commandAst.CommandElements | ForEach-Object { $_.ToString() }
  $cword = $elements.Count - 1
  $completions = wk __complete --cword $cword -- $elements
  foreach ($completion in $completions) {
    [System.Management.Automation.CompletionResult]::new($completion, $completion, 'ParameterValue', $completion)
  }
}
`
}
