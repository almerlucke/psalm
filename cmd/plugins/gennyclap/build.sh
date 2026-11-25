#!/bin/sh
# builds genny.clap (a shared library); copy it to ~/.clap for the host to find
cd "$(dirname "$0")" && go build -buildmode=c-shared -o genny.clap . && rm -f genny.h
