#!/usr/bin/env zsh

# Rewrite the old human-readable carry-forward marker in ledger text.
# Usage: ./scripts/migrate-since-markers.zsh [dunnit-root]

set -euo pipefail

ledger_root="${1:-${DUNNIT_DIR:-$HOME/.config/dunnit}}"
if [[ ! -d "$ledger_root" ]]; then
	print -u2 "Dunnit directory does not exist: $ledger_root"
	exit 1
fi

while IFS= read -r -d $'\0' file; do
	tmp="${file}.migrate-since.$$"
	sed -E 's/\(since ([0-9]{4}-[0-9]{2}-[0-9]{2})\)/s\/\1/g' "$file" >"$tmp"
	if cmp -s "$file" "$tmp"; then
		rm "$tmp"
	else
		mv "$tmp" "$file"
		print "updated $file"
	fi
done < <(find "$ledger_root" -type f -name '*.txt' -print0)
