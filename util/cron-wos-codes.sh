#!/usr/bin/env bash

# Export common binary paths so cron can locate `node` and other executables
export PATH="/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"

# Navigate to the correct directory inside your home folder
cd "$HOME/work/wos-giftcode-cli"

# Execute the pipeline
node js/code-scraper.js | dist/wos-client-darwin-arm64
