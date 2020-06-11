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

if ! [[ -d $dunnit_dir ]]; then
    echo "Creating fresh new dunnit dir: $dunnit_dir"
    mkdir -p $dunnit_dir
fi

if [[ -f $dunnit_file ]]; then
    last_update="LAST: $(sed -n '$p'  $dunnit_file | sed 's/^\[[0-9]*\] //')"
fi

dunnit-alert() {
    ans=$($alerter -reply \
		   -timeout 120 \
                   -title "Dunnit Activity Entry" \
		   -subtitle "What did you work on the last hour?" \
		   -closeLabel 'Nothing' \
		   -sound 'Glass' \
		   -message "${last_update}")

    if ! [[ -f $dunnit_file ]]; then
	echo "[$dt-$tm] Creating new dunnit file for today's work: $dunnit_file"
    fi

    # Bail out if user pressed 'Cancel'.
    if [[ $ans == 'Nothing' || $ans == '@TIMEOUT' ]]; then
	exit
    elif [[ $ans == '@ACTIONCLICKED' ]]; then
	# Support a SNOOZE hack by pressing 'Send' with an empty message.
	terminal-notifier -message 'Snoozing for 5m...'
	sleep 300
        # Recursive for snooze support!
	dunnit-alert
    fi

    tm=$(date +%H%M)
    echo "[$tm] $ans" >>$dunnit_file
    echo "[$dt-$tm] Captured your update in dunnit file: $dunnit_file"
}

dunnit-eod() {
    ans=$($alerter -timeout 120 \
                   -title "Dunnit Daily Summary" \
		   -message "Tag your day’s work??" \
		   -subtitle "You completed $(wc -l $dunnit_file | awk '{print $1}') today." \
		   -closeLabel 'Skip' \
		   -sound 'Glass'
		   )
    echo $ans
    if [[ $ans == '@ACTIONCLICKED' ]]; then
	echo "Firing up your editor on $dunnit_file"
	open -e $dunnit_file
    fi
}
