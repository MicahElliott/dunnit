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
                  # < <(ls $logfiles | sed -e 's#log/202./##' -e 's/\.log//' |tac) }

# See .envrc file for simpler Ctrl-T access to fzf
# FZF_CTRL_T_COMMAND='ls log/*/*/*.log | sed -e "s#log/202./##" -e "s/\.log//" |tac'
# FZF_CTRL_T_OPTS='--preview "bat --style=numbers --color=always log/$(date +%Y)/{}.log"'

# Micah likes this as a shortcut.
FZF_COMPLETION_TRIGGER=,

# Open logs that are retrieved from fzf Ctrl-T
alias -s log=$EDITOR
