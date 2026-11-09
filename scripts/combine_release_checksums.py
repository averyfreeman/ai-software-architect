# SPDX-FileCopyrightText: 2026 Leonardo Muffato (AUTOSOFT Engineering - www.autosoft-engineering.de)
# SPDX-License-Identifier: MIT

"""Create one checksum manifest for all target-specific release archives."""

from __future__ import annotations

import argparse
import hashlib
from collections.abc import Sequence
from pathlib import Path

CHUNK_SIZE = 1024 * 1024


def _sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        while chunk := stream.read(CHUNK_SIZE):
            digest.update(chunk)
    return digest.hexdigest()


def combine_checksums(archives: Sequence[Path], output: Path) -> Path:
    """Write sorted SHA-256 entries for the supplied release archives."""

    if not archives:
        raise ValueError("at least one release archive is required")

    resolved = [archive.resolve(strict=True) for archive in archives]
    names = [archive.name for archive in resolved]
    if len(names) != len(set(names)):
        raise ValueError("release archive filenames must be unique")
    if any(archive.suffix.casefold() != ".zip" for archive in resolved):
        raise ValueError("release checksum inputs must be ZIP archives")

    entries = sorted((archive.name, _sha256(archive)) for archive in resolved)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(
        "".join(f"{digest}  {name}\n" for name, digest in entries),
        encoding="ascii",
        newline="\n",
    )
    return output


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("archives", type=Path, nargs="+")
    args = parser.parse_args()
    combine_checksums(args.archives, args.output)
    print(f"Checksum manifest: {args.output}")


if __name__ == "__main__":
    main()
