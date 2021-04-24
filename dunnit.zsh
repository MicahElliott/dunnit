# this file is source'd by dunnit scripts and functions

# plist files have restricted paths
path+=/usr/local/bin

dt=$(date +%Y%m%d-%a) # 20200601-Mon
mo=$(gdate -dsunday +%b) # Month of nearest Sunday
wk=$(date +w%V-$mo) # w23-Jun
yr=$(date +%Y)
dunnit_dir=${DUNNIT_DIR:-~/dunnit/log/$yr/$wk}
dunnit_summary=$dunnit_dir/$dt.md
dunnit_ledger=~/dunnit/ledger-$dt.txt
dunnit_goals=~/dunnit/goals-$dt.txt
dunnit_tmp=$dunnit_dir/$dt-tmp.md

alerter=/usr/local/bin/alerter
# alerter=~/contrib/bin/alerter

if ! [[ -d $dunnit_dir ]]; then
    echo "Creating fresh new dunnit dir: $dunnit_dir"
    mkdir -p $dunnit_dir
fi

# Convert pieces of daily status working file into sections
sectionize-ledger() {
    # FIXME losing all non-tagged dunnits!
    # ggrep -q UNCOMPILED $dunnit_summary || { print "Already been compiled." 2>&1; return 1 }
    # gsed '/## Accomp/q' $dunnit_tmp 	# print only lines up to
    groups=( $(ggrep -E -o '#[0-9a-z]+' $dunnit_ledger | sort | uniq) )
    for g in $groups; do
	i2=$(sed 's/#//' <<<$g)
	print "\n### ${(C)i2}\n"
	ggrep $g $dunnit_ledger | sed -e "s/$g //" -e 's/^/- /'
        print '\n> IMPACT:'
    done
    print '\n### Other\n'
    ggrep -vE '#[0-9a-z]+|GOAL|TODO' $dunnit_ledger | gsed 's/^/- /'
    print '\n> IMPACT:'
    if ggrep -q ' TODO ' $dunnit_ledger; then
       print '\n### Incomplete\n'
       ggrep ' TODO ' $dunnit_ledger | gsed 's/^([A-Z]) TODO/-/'
    fi
}

maybe-create-ledger-file() {
    if ! [[ -f $dunnit_ledger ]]; then
	echo "[$dt-$tm] Creating new dunnit ledger file for today's work: $dunnit_ledger"
        touch $dunnit_ledger
    fi
}

# Create a new editable summary file from scratch
create-summary-file() {
    if [[ -f $dunnit_summary ]]; then
	print "Oops, summary file $dunnit_summary already exists."
    else
	echo "[$dt-$tm] Creating new dunnit summary file for today's work: $dunnit_summary"
	username=$(osascript -e "long user name of (system info)")
	# Create the file anew
        echo "# $username: Status $dt\n" >$dunnit_summary
        echo "**Sentiment:** (bad, neutral, or good)\n"  >>$dunnit_summary
	echo "**Summary:** (1 para)"  >>$dunnit_summary
	print '\n## Original Planned Goals\n' >>$dunnit_summary
	ggrep 'GOAL' $dunnit_ledger | gsed 's/^GOAL/-/' >>$dunnit_summary
        echo "\n## Accomplishments"  >>$dunnit_summary
	sectioned=$(sectionize-ledger)
	# [[ $? -eq 0 ]] || return 1
        echo $sectioned >> $dunnit_summary
	echo "\n## Biggest Thing of the Day\n"  >>$dunnit_summary
	echo "## Today I Learned\n"  >>$dunnit_summary
        # echo "\n## Plans/Problems\n" >>$dunnit_summary
    fi
}

dunnit-alert() {
    # Get most recent entry, prefer a TODO
    if [[ -f $dunnit_ledger ]]; then
	todo=$(tail -1 $dunnit_ledger | ggrep TODO)
	if [[ $? -ne 0 ]]; then
	    last_update="LAST: $(sed -n '$p' $dunnit_ledger | sed 's/^\[[0-9]*\] //')"
	else
	    last_update="TODO: $todo"
	fi
    fi

    if [[ -f /tmp/dunnit-nighty ]]; then
	echo 'in nighty mode; exiting as no-op'
	exit
    fi
    ans=$($alerter -reply \
		   -timeout 120 \
                   -title "Dunnit Activity Entry" \
		   -subtitle "Whadja work on? (blank to snooze)" \
		   -closeLabel 'Ignore' \
		   -sound 'Glass' \
		   -message "${last_update}")
    maybe-create-ledger-file
    # Bail out if user pressed 'Cancel'.
    if [[ $ans == 'Ignore' || $ans == '@TIMEOUT' ]]; then
	exit
    elif [[ $ans == '@ACTIONCLICKED' || $ans == 'snooze' || $ans == 'zzz' ]]; then
	# Support a SNOOZE hack by pressing 'Send' with an empty message.
	terminal-notifier -message 'Snoozing for 5m...'
	sleep 300
        # Recursive for snooze support!
	dunnit-alert
    fi

    if [[ $ans == 'Ditto' ]]; then
        # Determine if already a %N points indicator on last line
        if gsed -n '$p' $dunnit_ledger | ggrep -qE '%[0-9]+$'; then
	    # Increment
	    integer n=$(gsed -n '$p' $dunnit_ledger | gsed -r 's/.*%([0-9])$/\1/')
	    n+=1
            gsed -ri '$s/%[0-9]+/'"%$n/" $dunnit_ledger
	else
	    # Just append %1
            gsed -ri '$s/$/ %2/' $dunnit_ledger
        fi
	exit
    fi

    tm=$(gdate +%H:%M)
    if ggrep -q '^[A-Z]$' <<<$ans ; then
	item=$(ggrep "^($ans) " $dunnit_ledger | sed 's/([A-Z]) TODO //')
	[[ -z $item ]] && return 1
	gsed -i "/^($ans) /d" $dunnit_ledger # now remove the line
	echo "[$tm] $item" >>$dunnit_ledger
    else
	echo "[$tm] $ans" >>$dunnit_ledger
    fi
    echo "[$dt-$tm] Captured your update in dunnit file: $dunnit_ledger"
}

dunnit-editraw() {
    [[ -n $EDITOR ]] && $=EDITOR $dunnit_ledger  || open -e $dunnit_ledger &
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
    ans=$($alerter -timeout 120 \
                   -title "Dunnit Daily Summary" \
		   -message "Edit your day’s work (with tags etc)" \
		   -subtitle "You completed $(wc -l $dunnit_ledger | awk '{print $1}') today." \
		   -closeLabel 'Skip' \
		   -sound 'Glass')
    tm=$(gdate +%H:%M)
    if [[ $ans == '@ACTIONCLICKED' ]]; then
	if [[ -f $dunnit_summary ]]; then
	    print "Summary file already exists and may have been finessed already."
	    print "Will not overwrite."
	    exit 1
	else
            create-summary-file
	fi
	echo "[$dt-$tm] Opening editor on $dunnit_summary"
        # emacsclient --create-frame $dunnit_summary &
	[[ -n $EDITOR ]] && $=EDITOR $dunnit_summary  || open -e $dunnit_summary &
	# Open todoist instead
	# /usr/local/bin/cliclick kd:cmd,ctrl t:t ku:cmd,ctrl
    fi
}

dunnit-goals() {
    ans=$($alerter -reply \
	           -timeout 240 \
                   -title "Dunnit Daily Goals" \
		   -subtitle "Start your day with 3 goals." \
		   -message "Use Ctrl-Enter for extra goal lines." \
    		   -closeLabel 'Ignore' \
		   -sound 'Glass')
    # tm=$(gdate +%H:%M)
    if [[ $ans != "Ignore" ]]; then
       touch $dunnit_ledger
       gsed "s/$(echo -ne '\u2028')/\n/g" <<<$ans | gsed 's/^/GOAL /' >>$dunnit_ledger
    fi
    # emacsclient --create-frame $dunnit_summary &
    # [[ -n $EDITOR ]] && $=EDITOR $dunnit_summary  || open -e $dunnit_summary &
}

dunnit-report() {
    mkdir -p ~/dunnit/reports/
    # pandoc -f markdown $dunnit_summary -o ~/dunnit/reports/$dunnit_file:t:r.html
    html=$dunnit_summary:r.html
    pandoc -t html --self-contained --css reports/report.css -f markdown $dunnit_summary -o $html
    # pandoc -t html --self-contained --css reports/report.css -f markdown log/2021/w16-Apr/20210420-Tue.md -o foo.html
    # /Applications/Firefox.app/Contents/MacOS/firefox $html
    ${BROWSER-/Applications/Safari.app/Contents/MacOS/Safari} $html
}

dunnit-todo() {
    ans=$($alerter -reply \
	 	  -timeout 300 \
                  -title "Dunnit TODO" \
		  -subtitle "Whatcha gonna do next?" \
		  -message '\(just one thing)' \
                  -sound 'Glass')
    tm=$(gdate +%H:%M)
    if [[ $ans != '@CLOSED' ]]; then
	touch $dunnit_ledger
	# Generate a random letter
	# alpha=(); for c in {A..Z}; alpha+=$c; i=$(( RANDOM % 26 ))
	# Get highest letter todo
	if ! ggrep '^([A-Z]) TODO' $dunnit_ledger; then
	    next='A'
	else
	    next=$(ggrep TODO $dunnit_ledger | sort | tail -1 |
		    gsed -E -e 's/[()]//g' -e 's/ .*//' |
		    tr "0-9A-z" "1-9A-z_")
            # echo "($alpha[i]) TODO $ans" >>$dunnit_ledger
            echo "($next) TODO $ans" >>$dunnit_ledger
	    echo "[$dt-$tm] Captured your TODO in dunnit file: $dunnit_ledger"
	fi
    else
	echo no-op
    fi
}

dunnit-showtodos() {
    print '## Goals'
    ggrep 'GOAL' $dunnit_ledger | gsed 's/^GOAL/-/'
    print '\n## Todos'
    ggrep 'TODO' $dunnit_ledger | gsed 's/^TODO/-/'
}

dunnit-progress() {
    cat $dunnit_ledger
}
