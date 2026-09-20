#!/usr/bin/env python3
"""Run the deterministic M-16 retrieval benchmark."""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from tools.retrieval_benchmark.benchmark import main

if __name__ == "__main__":
    raise SystemExit(main())
