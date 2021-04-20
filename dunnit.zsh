# this file is source'd by dunnit scripts and functions

# plist files have restricted paths
path+=/usr/local/bin

dt=$(date +%Y%m%d-%a) # 20200601-Mon
mo=$(gdate -dsunday +%b) # Month of nearest Sunday
wk=$(date +w%V-$mo) # w23-Jun
yr=$(date +%Y)
dunnit_dir=${DUNNIT_DIR:-~/dunnit/log/$yr/$wk}
dunnit_file=$dunnit_dir/$dt.log

alerter=/usr/local/bin/alerter
# alerter=~/contrib/bin/alerter

if ! [[ -d $dunnit_dir ]]; then
    echo "Creating fresh new dunnit dir: $dunnit_dir"
    mkdir -p $dunnit_dir
fi

if [[ -f $dunnit_file ]]; then
    todo=$(tail -1 $dunnit_file | grep TODO)
    if [[ $? -ne 0 ]]; then
	last_update="LAST: $(sed -n '$p' $dunnit_file | sed 's/^\[[0-9]*\] //')"
    else
	last_update="TODO: $todo"
    fi
fi

dunnit-alert() {
    if [[ -f /tmp/dunnit-nighty ]]; then
	echo 'in nighty mode'
	exit
    fi
    ans=$($alerter -reply \
		   -timeout 120 \
                   -title "Dunnit Activity Entry" \
		   -subtitle "Whadja work on? (blank to snooze)" \
		   -closeLabel 'Ignore' \
		   -sound 'Glass' \
		   -message "${last_update}")

    if ! [[ -f $dunnit_file ]]; then
	echo "[$dt-$tm] Creating new dunnit file for today's work: $dunnit_file"
	username=$(osascript -e "long user name of (system info)")
	echo "# $username: Status for $dt\n" >$dunnit_file
	echo "**Summary:**\n"  >>$dunnit_file
	echo "## Accomplishments\n"  >>$dunnit_file
    fi

    # Bail out if user pressed 'Cancel'.
    if [[ $ans == 'Ignore' || $ans == '@TIMEOUT' ]]; then
	exit
    elif [[ $ans == '@ACTIONCLICKED' ]]; then
	# Support a SNOOZE hack by pressing 'Send' with an empty message.
	terminal-notifier -message 'Snoozing for 5m...'
	sleep 300
        # Recursive for snooze support!
	dunnit-alert
    fi

    tm=$(gdate +%H:%M)
    # Even if ans was DONE, record it as such
    echo "- $ans [$tm]" >>$dunnit_file
    echo "[$dt-$tm] Captured your update in dunnit file: $dunnit_file"
}

dunnit-alert-todoist() {
    if [[ -f /tmp/dunnit-nighty ]]; then
	echo 'in nighty mode'
	exit
    fi
    terminal-notifier -sound Glass -message 'Whadja work on?' -title 'Dunnit Reminder'
    # Pop up fast-entry for todoist
    /usr/local/bin/cliclick kd:cmd,ctrl t:a ku:cmd,ctrl
}

dunnit-eod() {
    set -x
    ans=$($alerter -timeout 120 \
                   -title "Dunnit Daily Summary" \
		   -message "Edit your day’s work (with tags etc)??" \
		   -subtitle "You completed $(wc -l $dunnit_file | awk '{print $1}') today." \
		   -closeLabel 'Skip' \
		   -sound 'Glass')
    tm=$(gdate +%H%M)
    if [[ $ans == '@ACTIONCLICKED' ]]; then
	echo "[$dt-$tm] Opening editor on $dunnit_file"
	echo "\n## Problems\n" >>$dunnit_file
	echo "## Plans\n" >>$dunnit_file
	# open -e $dunnit_file # can't choose arbitrary editor with finder
        emacsclient --create-frame $dunnit_file &
	# Open todoist instead
	# /usr/local/bin/cliclick kd:cmd,ctrl t:t ku:cmd,ctrl
    fi
    set +x
}

dunnit-todo() {
    ans=$(alerter -reply \
		  -timeout 300 \
                  -title "Dunnit TODO" \
		  -subtitle "Whatcha gonna do next?" \
		  -message '\(just one thing)' \
                  -sound 'Glass')
    tm=$(gdate +%H:%M)
    if [[ $ans != '@CLOSED' ]]; then
	echo "[$dt-$tm] Got it"
	echo "[$tm] TODO $ans" >>$dunnit_file
	echo "[$dt-$tm] Captured your TODO in dunnit file: $dunnit_file"
    else
	echo no-op
    fi
}
