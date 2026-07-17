# Automated Gift-Code Redemption Setup

This guide describes how to configure the client pipeline to run automatically once a day using `cron`.

The configuration expects the repository to be checked out at `$HOME/work/wos-giftcode-cli` with the runner script located directly above it at `$HOME/work/cron-wos-codes.sh`.

### Directory Structure Layout

```text
$HOME/                      # Your user home directory 
├── .wos-cli-env            # Global config file (defines DEFAULT_FID)
├── .wos-redeemed-codes.json# Cache tracking successfully redeemed codes
└── work/                   # Parent work directory
    ├── cron-wos-codes.sh   # The executed runner script (copied in Step 1)
    └── wos-giftcode-cli/   # Checked-out repository folder
        ├── dist/
        │   └── wos-client-darwin-arm64 # Precompiled Go client binary
        └── js/
            └── code-scraper.js         # Gift-code scraper script
```

---

## Installation Steps

### Step 1: Copy the Runner Script
Copy the runner script from the repository's `util/` directory to the `$HOME/work/` parent directory:
```bash
cp util/cron-wos-codes.sh $HOME/work/cron-wos-codes.sh
```

### Step 2: Make the Script Executable
Give the script execution permissions:
```bash
chmod +x $HOME/work/cron-wos-codes.sh
```

### Step 3: Configure Your Player ID
Since cron runs completely non-interactively, your player ID must be configured in your global home environment config:
```bash
echo "DEFAULT_FID=YOUR_PLAYER_ID" > ~/.wos-cli-env
```
*(Replace `YOUR_PLAYER_ID` with your actual Whiteout Survival player ID)*

### Step 4: Schedule with Crontab
Open your crontab configuration editor:
```bash
crontab -e
```

Paste the sample entry from `util/crontab.sample` (or write this line) to execute the script daily at midnight (12:00 AM) and redirect output/errors to a log file:
```text
0 0 * * * $HOME/work/cron-wos-codes.sh > /tmp/wos-cron.log 2>&1
```

Save and exit the editor.

---

## Verifying Setup
To verify the cron setup is working correctly:
1. Run the script manually to confirm it succeeds without errors:
   ```bash
   $HOME/work/cron-wos-codes.sh
   ```
2. Inspect the log file `/tmp/wos-cron.log` to check the execution output.
