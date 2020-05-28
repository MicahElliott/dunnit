#! /bin/zsh

dt=$(date +%Y%m%d)
tm=$(date +%H%M)
dunnit_dir=${DUNNIT_DIR:-~/doc/dunnit}
dunnit_file=$dunnit_dir/$dt.log

stamp="[$tm]"

if ! [[ -d $dunnit_dir ]]; then
   echo "Creating fresh new dunnit dir: $dunnit_dir"
   mkdir -p $dunnit_dir
fi

if [[ -f $dunnit_file ]]; then
    last_update="LAST: $(sed -n '$p'  $dunnit_file | sed 's/^\[[0-9]*\] //')"
fi

ans=$(/usr/local/bin/alerter -reply -timeout 120 -sound default \
      -title "Dunnit..." \
      -subtitle "What did you work on the last hour?" \
      -message "${last_update}")

if ! [[ -f $dunnit_file ]]; then
    echo "Creating new dunnit file for today's work: $dunnit_file"
fi

# Bail out if user pressed 'Cancel'.
if [[ $ans == '@CLOSED' || $ans == '@TIMEOUT' ]]; then exit; fi

line="$stamp $ans"

echo $line >>$dunnit_file

echo "Captured your update in dunnit file: $dunnit_file"
