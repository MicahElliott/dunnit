# this file is source'd by dunnit scripts and functions

# plist files have restricted paths
path+=/usr/local/bin

username=$(osascript -e "long user name of (system info)")
dt=$(gdate +%Y%m%d-%a) # 20200601-Mon
mo=$(gdate -dsunday +%b) # Month of nearest Sunday
wk=$(gdate +w%V-$mo) # w23-Jun
yr=$(gdate +%Y)
if [[ $(gdate +%a) == 'Mon' ]]; then
    dunnit_yesterday=$(gdate -d 'last friday' +%Y%m%d-%a)
    dunnit_ledger_yesterday=~/dunnit/ledger-$dunnit_yesterday.txt
else
    dunnit_yesterday=$(gdate -d 'yesterday' +%Y%m%d-%a)
    dunnit_ledger_yesterday=~/dunnit/ledger-$dunnit_yesterday.txt
fi
dunnit_dir=${DUNNIT_DIR:-~/dunnit/log/$yr/$wk}
dunnit_summary=$dunnit_dir/$dt.md
# dunnit_ledger=~/dunnit/ledger-$dt.txt
dunnit_ledger=$dunnit_dir/ledger-$dt.txt
# dunnit_goals=~/dunnit/goals-$dt.txt
dunnit_tmp=$dunnit_dir/$dt-tmp.md
dunnit_nighty=/tmp/dunnit-nighty

dunnit_browser=${BROWSER-/Applications/Safari.app/Contents/MacOS/Safari}
alerter=/usr/local/bin/alerter
# alerter=~/contrib/bin/alerter

if ! [[ -d $dunnit_dir ]]; then
    echo "Creating fresh new dunnit dir: $dunnit_dir"
    mkdir -p $dunnit_dir
fi

dunnit-edit() {
    [[ -n $EDITOR ]] && $=EDITOR $1 || open -e $1 &
}

dunnit-editraw() {
    dunnit-edit $dunnit_ledger
}

get-bullets() {
    local tag=$1
    ggrep -vE 'GOAL|TODO' $dunnit_ledger | ggrep $tag |
	gsed -r -e "s/$tag //" -e 's/^/- /' \
	     -e 's/ #[0-9a-z]+//g' \
	     -e 's/ \[[0-9:]+\] / /'
}

# Convert pieces of daily status working file into sections
sectionize-ledger() {
    # FIXME losing all non-tagged dunnits!
    # ggrep -q UNCOMPILED $dunnit_summary || { print "Already been compiled." 2>&1; return 1 }
    # gsed '/## Accomp/q' $dunnit_tmp 	# print only lines up to
    groups=( $(ggrep -vE 'GOAL|TODO' $dunnit_ledger |
		   ggrep -E -o '#[0-9a-z]+' | sort | uniq) )
    integer gcount=1
    for g in $groups; do
	i2=$(sed 's/#//' <<<$g)
	items=$(get-bullets $g)
	integer item_count=$(wc -l <<<$items)
	print "\n## ${(C)i2} ($item_count)\n"
	print -- $items
	stmt=$impact_statements[$gcount]
	print "\n> IMPACT-$stmt"
        # print '\n> IMPACT(XXX):'
	# Multiple impacts if many items
	# (( item_count > 3 )) && print '\n> IMPACT(XXX):'
	gcount+=1
    done
    ggrep -qvE '#[0-9a-z]+|GOAL|TODO' $dunnit_ledger && print '\n## Other\n'
    ggrep  -vE '#[0-9a-z]+|GOAL|TODO' $dunnit_ledger | gsed -r -e 's/^/- /'  -e 's/ \[[0-9:]+\] / /'
    # print '\n> IMPACT:'
    if ggrep -q ' TODO ' $dunnit_ledger; then
       print '\n## Incomplete\n'
       ggrep ' TODO ' $dunnit_ledger | gsed 's/^([A-Z]) /- /'
       # TODO Carry over tomorrow's goals
       print '\n## Tomorrow’s Goals\n'
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
        # Create the file anew
        echo "% $username" >$dunnit_summary
        echo "% Impact Report" >>$dunnit_summary
        echo "% $dt\n" >>$dunnit_summary
	# print -- '<!-- See instructions at end of file. They’ll be automatically removed for you, as will this section. -->\n' >>$dunnit_summary
	echo "# Overview\n"  >>$dunnit_summary
	echo "### Sentiment: $sentiment\n"  >>$dunnit_summary
	echo '## Summary\n' >>$dunnit_summary
	# print -- '\n<!-- Write one short paragraph here summarizing the day. -->\n' >>$dunnit_summary
	print "$summary" >>$dunnit_summary
        # print 'XXX' >>$dunnit_summary
	print '\n## 🥅 Original Goals 🥅\n' >>$dunnit_summary
	ggrep 'GOAL' $dunnit_ledger | gsed 's/^GOAL/-/' >>$dunnit_summary
        echo "\n# Accomplishments"  >>$dunnit_summary
	# print -- '\n<!-- Combine bullets for each section into fewer and add a summary impact description and scores (replace XXX). -->' >>$dunnit_summary
        echo "\n### Productivity Score: $productivity"  >>$dunnit_summary
	sectioned=$(sectionize-ledger)
	# [[ $? -eq 0 ]] || return 1
        echo $sectioned >> $dunnit_summary
        # echo "\n# Other"  >>$dunnit_summary
	# echo "\n## 🎉 Highlight\n"  >>$dunnit_summary
	# echo "## Today I Learned\n"  >>$dunnit_summary
        # echo "\n## Plans/Problems\n" >>$dunnit_summary
	print '\n---- DELETE_TO_EOF ----'  >>$dunnit_summary
	[[ -z $1 ]] && cat ~/dunnit/summary-instructions.md >>$dunnit_summary
    fi
}

dunnit-alert() {
    set -x
    if [[ $1 != 'frommenu' ]]; then
	# Don't pop if recently shown (15m)
	# OK if empty since will be midnight default
	stamp=$(ggrep -E '\[[0-9:]+' $dunnit_ledger | sort -n | tail -1 | gsed -r 's/\[([0-9:]+)\] .*/\1/g')
	secs_last=$(gdate -d "$stamp" +%s)
	secs_now=$(gdate +%s)
	if (( (secs_now - secs_last) / 60 < 15 )); then
	    print 'Not popping since recently shown'
	    exit
	fi
    fi

    # Get most recent entry, prefer a TODO
    if [[ -f $dunnit_ledger ]]; then
	todo=$(tail -1 $dunnit_ledger | ggrep 'TODO ')
	if [[ $? -ne 0 ]]; then
	    last_update="LAST: $(sed -n '$p' $dunnit_ledger | sed 's/^\[[0-9]*\] //')"
	else
	    last_update="TODO: $todo"
	fi
    fi

    if [[ -f $dunnit_nighty && $1 != 'frommenu' ]]; then
	echo 'in nighty mode; exiting as no-op'
	exit
    fi

    # Gather and combine tag suggestions from these prescriptions,
    # plus what's already been used today (should expand to week).
    # Cool thing about alerter is that when you open it to reply, it
    # puts the end of the long text into view.
    suggs=( '#til' '#mtg' '#question' '#nts' '#personal' )
    suggs+=( $(ggrep -E -o '#[0-9a-z]+' $dunnit_ledger) )
    suggs=$(print -l $suggs | sort | uniq)
    todos=$(ggrep ' TODO ' $dunnit_ledger)
    ans=$($alerter -reply \
		   -appIcon ~/dunnit/dunnit-icon-green.png \
		   -timeout 600 \
                   -closeLabel 'Ignore' \
		   -sound 'Glass' \
                   -title "Dunnit Activity Entry" \
		   -subtitle "What did you work on? (blank to snooze)" \
                   -message "${last_update} — TODOs: $todos — Tags: $suggs")
    maybe-create-ledger-file
    # Bail out if user pressed 'Cancel'.
    if [[ $ans == 'Ignore' || $ans == '@TIMEOUT' ]]; then
	exit
    elif [[ $ans == '@ACTIONCLICKED' || $ans == 'snooze' || $ans == 'zzz' ]]; then
	# Support a SNOOZE hack by pressing 'Send' with an empty message.
	terminal-notifier -appIcon ~/dunnit/dunnit-icon-yellow.png \
			  -message 'Snoozing for 5m...'
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

    if [[ -z $ans ]]; then
	terminal-notifier -sound Glass -title 'MAC BUG: UNABLE TO CAPTURE' -message 'Close this and try again manually.'
	exit
    fi

    tm=$(gdate +%H:%M)
    if ggrep -q '^[A-Z]$' <<<$ans ; then
	item=$(ggrep "^($ans) " $dunnit_ledger | sed 's/([A-Z]) TODO //')
	[[ -z $item ]] && return 1
	gsed -i "/^($ans) /d" $dunnit_ledger # now remove the line
	echo "[$tm] $item" >>$dunnit_ledger
    else
	ans=$(gsed "s/$(print -n '\u2028')/\n[$tm] /g" <<<$ans)
	echo "[$tm] $ans" >>$dunnit_ledger
    fi
    echo "[$dt-$tm] Captured your update in dunnit file: $dunnit_ledger"
    if ! ggrep -q '#' <<<$ans; then
	terminal-notifier -title 'Did you know you can use "tags"?' \
			  -appIcon ~/dunnit/dunnit-icon-yellow.png \
			  -subtitle 'They help with categorizing your daily report.' \
			  -message 'Eg: Mowed the lawn #chore'
    fi
    set +x
}

dunnit-alert-todoist() { 	# NIU
    if [[ -f /tmp/dunnit-nighty ]]; then echo 'in nighty mode'; exit; fi
    terminal-notifier -sound Glass -message 'Whadja work on?' -title 'Dunnit Reminder'
    /usr/local/bin/cliclick kd:cmd,ctrl t:a ku:cmd,ctrl
}

dunnit-nighty-off() { [[ -f $dunnit_nighty ]] && rm $dunnit_nighty }
dunnit-nighty-on()  { touch $dunnit_nighty }

dunnit-bod() {
    dunnit-goals
    dunnit-nighty-off
}

dunnit-eod() {
    ans=$($alerter -timeout 600 \
		   -sound 'Glass' \
                   -appIcon ~/dunnit/dunnit-icon-yellow.png \
	           -title "Dunnit Daily Summary" \
		   -subtitle "You completed $(ggrep -cE '\[[0-9:]+\]' $dunnit_ledger) today." \
		   -message "Finalize your day’s work (#tags etc)" \
		   -actions 'Finalize' \
		   -closeLabel 'Too lazy today')
    tm=$(gdate +%H:%M)
    if [[ $ans == 'Finalize' ]]; then
	if [[ -f $dunnit_summary ]]; then
	    print "Summary file already exists and may have been finessed already."
	    print "Will not overwrite."
	    exit 1
	else
	    summary=$(
		alerter -reply \
                  	-timeout 600 \
			-appIcon ~/dunnit/dunnit-icon-yellow.png \
                        -title 'Dunnit Wrap-Up' \
			-message "Summarize your day in a sentence or two.")
	    productivity=$(
		alerter -title 'Dunnit Wrap-Up' \
                        -timeout 600 \
			-appIcon ~/dunnit/dunnit-icon-yellow.png \
			-message "How productive was your day?" \
			-actions 1,2,3,4,5 \
			-dropdownLabel 'Score it!')
	    sentiment=$(
		alerter -title 'Dunnit Wrap-Up' \
			-timeout 600 \
			-appIcon ~/dunnit/dunnit-icon-yellow.png \
			-message 'What was your sentiment for the day?' \
			-actions 'Bad,Neutral,Good' \
			-dropdownLabel 'Rate it!')
	    groups=( $(ggrep -vE 'GOAL|TODO' $dunnit_ledger |
			   ggrep -E -o '#[0-9a-z]+' | sort | uniq) )
            terminal-notifier -title 'Dunnit Wrap-Up' \
			      -appIcon ~/dunnit/dunnit-icon-yellow.png \
			      -subtitle 'Let’s review your tag sections.' \
			      -message 'Give a brief impact statement for each tag section.'
	    impact_statements=()
	    set -x
	    for g in $groups; do
		# Remove first dash so alerter doesn't blow up
		bullets=$(get-bullets $g | gsed -r '1s/^- //')
		impact_statements+=$(
		    alerter -reply \
			    -appIcon ~/dunnit/dunnit-icon-yellow.png \
			    -timeout 600 \
			    -title 'Dunnit Wrap-Up' \
			    -subtitle "What was the impact of $g?" \
			    -message "$bullets")
			    # -message 'Ex: 3: Team can integrate the kW validation now.')
	    done
	    set +x
            create-summary-file
	fi
	echo "[$dt-$tm] Opening editor on $dunnit_summary"
        # emacsclient --create-frame $dunnit_summary &
	dunnit-edit $dunnit_summary
        # Open todoist instead
	# /usr/local/bin/cliclick kd:cmd,ctrl t:t ku:cmd,ctrl
    fi
    dunnit-nighty-on
}

dunnit-autofinalize() {
    create-summary-file noinstructions
    # TODO Maybe gen report, open browser, send email, etc
}

dunnit-goals() {
    touch $dunnit_ledger
    dunnit-nighty-off
    if ggrep -q GOAL $dunnit_ledger; then
	print 'Already set goals today'
        exit
    fi
    ans=$($alerter -reply \
		   -appIcon ~/dunnit/dunnit-icon-purple.png \
	           -timeout 600 \
                   -title "Dunnit Daily Goals" \
		   -subtitle "Start your day with 3 high-level goals." \
		   -message "Use Ctrl-Return for each new goal line." \
    		   -closeLabel 'Ignore' \
		   -sound 'Glass')
    # tm=$(gdate +%H:%M)
    [[ $ans == 'Ignore' || $ans == '@TIMEOUT' ]] && exit

    gsed "s/$(print -n '\u2028')/\n/g" <<<$ans | gsed 's/^/GOAL /' >>$dunnit_ledger
    # Carry yesterday's unfinished TODOs and BLOCKERS into today
    ggrep 'TODO ' $dunnit_ledger_yesterday >>$dunnit_ledger
    ggrep 'BLOCKER ' $dunnit_ledger_yesterday >>$dunnit_ledger
    terminal-notifier -title 'Dunnit Confirmation' \
		      -appIcon ~/dunnit/dunnit-icon-purple.png \
		      -subtitle 'Sounds great!' \
		      -message 'You’re set up for a successful day!'

    # emacsclient --create-frame $dunnit_summary &
    # [[ -n $EDITOR ]] && $=EDITOR $dunnit_summary  || open -e $dunnit_summary &
}

dunnit-report() {
    set -x
    gsed -ir '/---- DELETE_TO_EOF /,$d' $dunnit_summary
    print 'Saving your day’s work'
    dunnit-push
    mkdir -p ~/dunnit/reports/
    # pandoc -f markdown $dunnit_summary -o ~/dunnit/reports/$dunnit_file:t:r.html
    # html=$dunnit_summary:r-report.html
    html=~/dunnit/reports/$dt-report.html
    # preso=$dunnit_summary:r-preso.html
    preso=~/dunnit/reports/$dt-preso.html
    pandoc -t html --self-contained --css ~/dunnit/reports/report.css -f markdown $dunnit_summary -o $html
    pandoc -s -t revealjs $dunnit_summary -o $preso
    # pandoc -t html --self-contained --css reports/report.css -f markdown log/2021/w16-Apr/20210420-Tue.md -o foo.html
    # /Applications/Firefox.app/Contents/MacOS/firefox $html
    $dunnit_browser $html
    $dunnit_browser $preso
    set +x
}

dunnit-todo() {
    ans=$($alerter -reply \
		   -appIcon ~/dunnit/dunnit-icon-blue.png \
	 	   -timeout 300 \
                   -title "Dunnit TODO" \
		   -subtitle "What will you do next?" \
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
	    next=$(ggrep 'TODO ' $dunnit_ledger | sort | tail -1 |
		    gsed -E -e 's/[()]//g' -e 's/ .*//' |
		    gtr "0-9A-z" "1-9A-z_")
            # echo "($alpha[i]) TODO $ans" >>$dunnit_ledger
	fi
        print "($next) TODO $ans" >>$dunnit_ledger
	echo "[$dt-$tm] Captured your TODO in dunnit file: $dunnit_ledger"
    else
	echo no-op
    fi
}

dunnit-blocker() {
    ans=$($alerter -reply \
		   -appIcon ~/dunnit/dunnit-icon-red.png \
		   -timeout 300 \
                   -title "Dunnit Blocker/Question" \
		   -subtitle "What are you hung up on?")
    if [[ $ans != '@CLOSED' ]]; then
        print "BLOCKER $ans" >>$dunnit_ledger
	echo "[$dt-$tm] Captured your BLOCKER in dunnit file: $dunnit_ledger"
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

dunnit-lunchtime() {
    if ggrep -q GOAL $dunnit_ledger; then
	ans=$($alerter -timeout 3000 \
		       -sound Glass \
		       -title 'Dunnit Goals Reminder' \
		       -appIcon ~/dunnit/dunnit-icon-orange.png \
		       -subtitle 'How are your goals coming along?' \
                       -message "Click: Dunnit -> Planning -> View All" \
		       -actions 'GREAT!! 🔥' \
		       -closeLabel '😢')
    else
	terminal-notifier -sound Glass \
			  -title 'Dunnit Goals Reminder' \
			  -appIcon ~/dunnit/dunnit-icon-red.png \
			  -subtitle 'Umm, you have no goals set for today 😢'
	dunnit-goals
    fi
}

dunnit-progress() {
    cat $dunnit_ledger
}

dunnit-pomodoro() {
    ans=$($alerter -reply \
		   -appIcon ~/dunnit/dunnit-icon-brown.png \
	 	   -timeout 300 \
                   -title "Dunnit New Activity" \
		   -subtitle "Time and create a new running Dunnit" \
		   -message 'Ex: 25m Build reactor monitor #fission' \
                   -sound 'Glass')
    set -x
    if ! ggrep -q '^@' <<<$ans; then
	print $ans
	duration=$(gsed -r 's/([0-9]+m) .*/\1/' <<<$ans)
	task=$(gsed -r 's/^[0-9]+m //' <<<$ans)
        print -- $duration
	print -- $task
	sleep $duration
        terminal-notifier -sound Glass -title 'Dunnit Activity Time Over' \
			  -appIcon ~/dunnit/dunnit-icon-brown.png \
			  -subtitle 'We’re back! Recording as done.' \
			  -message "$task"
	tm=$(gdate +%H:%M)
	print "[$tm] $task" >>$dunnit_ledger
    fi
    set +x
}

dunnit-push() {
    set -x
    print "Adding/committing/pushing all changes to mydunnits remote."
    cd ~/dunnit/log && {
	git add -A .
	git commit -am "checkpoint sync from $HOST"
	git push origin master
    }
    set +x
}

dunnit-pull() {
    set -x
    print "Committing and pulling all changes from remote mydunnits repo."
    cd ~/dunnit/log && {
	git add -A .
	git commit -am "checkpoint sync from $HOST"
	git pull --rebase
    }
    set +x
}

dunnit-preferences() {
    set -x
    dunnit-edit ~/dunnit/config.zsh
    source ~/dunnit/config.zsh
    dunnit_cfg=~/dunnit/config-templates
    if [[ -f $dunnit_cfg/dunnit-standup.plist &&
	  -n $DUNNIT_STANDUP ]]; then
	local hour=$DUNNIT_STANDUP[1] min=$DUNNIT_STANDUP[2]
	gsed -r -e "s/STANDUP_HOUR/$hour/" -e "s/STANDUP_MINUTE/$min/" \
	     $dunnit_cfg/dunnit-standup.plist \
	     >|~/dunnit/dunnit-standup.plist
	print "Reloading standup scheduler for time: $DUNNIT_STANDUP"
	launchctl load -w ~/dunnit/dunnit-standup.plist
    fi
    set +x
}

dunnit-standup() {
    print 'Time for standup!'
    $dunnit_browser ~/dunnit/reports/$dunnit_yesterday-report.html
}
