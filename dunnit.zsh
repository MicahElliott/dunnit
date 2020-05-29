#! /bin/zsh

dt=$(date +%Y%m%d)
dunnit_dir=${DUNNIT_DIR:-~/doc/dunnit}
dunnit_file=$dunnit_dir/$dt.log

alerter=/usr/local/bin/alerter

if ! [[ -d $dunnit_dir ]]; then
    echo "Creating fresh new dunnit dir: $dunnit_dir"
    mkdir -p $dunnit_dir
fi

if [[ -f $dunnit_file ]]; then
    last_update="LAST: $(sed -n '$p'  $dunnit_file | sed 's/^\[[0-9]*\] //')"
fi

alert() {
    ans=$($alerter -reply \
		   -timeout 120 \
		   -sound default \
		   -title "Dunnit..." \
		   -subtitle "What did you work on the last hour?" \
		   -closeLabel 'Nothing' \
		   -sound 'Glass' \
		   -message "${last_update}")

    if ! [[ -f $dunnit_file ]]; then
	echo "Creating new dunnit file for today's work: $dunnit_file"
    fi

    # Bail out if user pressed 'Cancel'.
    if [[ $ans == '@CLOSED' || $ans == '@TIMEOUT' ]]; then
	exit
    elif [[ $ans == '@ACTIONCLICKED' ]]; then
	# Support a SNOOZE hack by pressing 'Send' with an empty message.
	terminal-notifier -message 'Snoozing for 5m...'
	sleep 300
        # Recursive!
	alert
    fi

    tm=$(date +%H%M)
    stamp="[$tm]"
    line="$stamp $ans"
    echo $line >>$dunnit_file
    echo "Captured your update in dunnit file: $dunnit_file"
    exit
}
alert
