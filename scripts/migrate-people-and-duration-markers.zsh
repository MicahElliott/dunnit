#!/usr/bin/env zsh

# Migrate the original trackable markers in ledger text:
#   &Person -> @Person
#   @20m   -> ~20m
#
# Usage: ./scripts/migrate-people-and-duration-markers.zsh [dunnit-root]
# Run this once after updating Dunnit to the matching parser/UI format.

set -euo pipefail

ledger_root="${1:-${DUNNIT_DIR:-$HOME/.config/dunnit}}"
if [[ ! -d "$ledger_root" ]]; then
	print -u2 "Dunnit directory does not exist: $ledger_root"
	exit 1
fi

while IFS= read -r -d $'\0' file; do
	tmp="${file}.migrate-trackables.$$"
	perl -pe '
		s{(^|[^[:alnum:]_])@([0-9]+[mhd])(?=$|[[:space:]])}{$1~$2}g;
		s{(^|[^[:alnum:]_])&([[:alpha:]][[:alnum:]_-]*)}{$1\@$2}g;
	' "$file" >"$tmp"
	if cmp -s "$file" "$tmp"; then
		rm "$tmp"
	else
		mv "$tmp" "$file"
		print "updated $file"
	fi
done < <(find "$ledger_root" -type f -name 'ledger-*.txt' -print0)
