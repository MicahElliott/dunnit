### FZF Completion
# Enable completion of log files
# % dunnit ,<TAB>
# > log/20200530.log
# > ...
_fzf_complete_dunnit() {
    logfiles=( log/*/*/*.log )
    _fzf_complete --multi  \
                  --preview 'bat -l log --color always log/$(date +%Y)/{}.log' \
                  --prompt="log> " -- "$@" \
                  < <(ls $logfiles | sed -e 's#log/202./##' -e 's/\.log//' |tac) }
                  # < <( for f in $logfiles; sed -e 's#.*/log-##' -e 's/\..*//' <<<$f ) }

# Micah likes this as a shortcut.
FZF_COMPLETION_TRIGGER=,
