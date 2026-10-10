#!/usr/bin/env bash
# Installation script for WOS Captcha Solver (ddddocr + OpenCV)
# Supported OS: macOS, Linux (Unix environments)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SOLVER_DIR="${SCRIPT_DIR}/solver"
VENV_DIR="${SOLVER_DIR}/venv"

echo "=== WOS CAPTCHA Solver Setup ==="

# 1. Verify python3 is installed
if ! command -v python3 >/dev/null 2>&1; then
    echo "Error: python3 was not found on your system." >&2
    echo "Please install Python 3.8+ and try again." >&2
    exit 1
fi

echo "Found Python: $(python3 --version)"

# 2. Check for virtual environment tool (venv)
if ! python3 -m venv --help >/dev/null 2>&1; then
    echo "Error: The Python 'venv' module is missing." >&2
    echo "Please install python3-venv (on Linux) or ensure your Python setup supports virtual environments." >&2
    exit 1
fi

# 3. Create virtual environment
if [ -d "$VENV_DIR" ]; then
    echo "Virtual environment already exists in ${VENV_DIR}. Updating it..."
else
    echo "Creating virtual environment in ${VENV_DIR}..."
    python3 -m venv "$VENV_DIR"
fi

# 4. Determine virtual environment Python and Pip binaries
VENV_PYTHON="${VENV_DIR}/bin/python"
VENV_PIP="${VENV_DIR}/bin/pip"

# 5. Upgrade pip
echo "Upgrading pip..."
"$VENV_PIP" install --upgrade pip

# 6. Install dependencies
echo "Installing dependencies (opencv-python, ddddocr) inside virtual environment..."
if [ -f "${SOLVER_DIR}/requirements.txt" ]; then
    "$VENV_PIP" install -r "${SOLVER_DIR}/requirements.txt"
else
    "$VENV_PIP" install opencv-python ddddocr
fi

# 7. Verification dry-run
echo "Verifying installation..."
if "$VENV_PYTHON" -c "import cv2; import ddddocr; print('Verification: ddddocr and cv2 imported successfully!')" >/dev/null 2>&1; then
    echo ""
    echo "=========================================================="
    echo "SUCCESS: WOS CAPTCHA Solver Setup completed successfully!"
    echo "=========================================================="
    echo "The Go and JS clients will now automatically detect"
    echo "and use this local solver in the background!"
    echo "No manual action required when running the clients."
    echo "=========================================================="
else
    echo "Error: Verification failed. Dependencies could not be imported correctly." >&2
    exit 1
fi
