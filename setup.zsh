### FZF Completion
# Enable completion of sqlstash queries.
# % dunnit ,<TAB>
# > log/20200530.log
# > ...
_fzf_complete_sqlstash() {
    qfiles=( log/*.log)
    _fzf_complete --multi --reverse \
                  --preview 'bat -l sql --color always queries/query-{}.tpl.sql' \
                  --prompt="log> " -- "$@" \
                  < <( for f in $logfiles; sed -e 's#.*/log-##' -e 's/\..*//' <<<$f ) }

# Micah likes this as a shortcut.
FZF_COMPLETION_TRIGGER=,
