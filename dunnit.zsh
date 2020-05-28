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

ans=$(alerter -reply -timeout 120 -sound default -message "What did you work on the last hour?" -title "Dunnit...")

if ! [[ -f $dunnit_file ]]; then
    echo "Creating new dunnit file for today's work: $dunnit_file"
fi

line="$stamp $ans"

echo $line >>$dunnit_file

echo "Captured your update in dunnit file: $dunnit_file"
