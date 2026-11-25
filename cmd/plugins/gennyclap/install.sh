#!/bin/sh
# installs genny.clap in ~/.clap: written next to the old one and moved over it, so a copy that a host has loaded is not
# written into (that can crash it). Quit the host first: it keeps the old copy until it is started again
set -e
dir="${CLAP_DIR:-$HOME/.clap}"
src="$(dirname "$0")/genny.clap"
mkdir -p "$dir"
cp "$src" "$dir/genny.clap.new"
mv -f "$dir/genny.clap.new" "$dir/genny.clap"
echo "installed $dir/genny.clap ($(date +%H:%M:%S)); start the host again"
