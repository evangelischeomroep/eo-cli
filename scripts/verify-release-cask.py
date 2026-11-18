"""Match a generated Cask to the published release checksum manifest."""

import re
import sys
from pathlib import Path


def verify(tag, cask, checksums):
    if re.findall(r'^  version "([^"]+)"$', cask, re.MULTILINE) != [tag[1:]]:
        raise ValueError(f"Cask version does not match {tag}")

    archives = {
        f"eo_{system}_{arch}.tar.gz"
        for system in ("darwin", "linux")
        for arch in ("amd64", "arm64")
    }
    pairs = re.findall(
        r'^\s+sha256 "([0-9a-f]{64})"\n'
        r'\s+url "https://github\.com/evangelischeomroep/eo-cli/releases/download/'
        r'v#\{version\}/([^"]+)"',
        cask,
        re.MULTILINE,
    )
    if len(pairs) != len(archives) or {name for _, name in pairs} != archives:
        raise ValueError("Cask must reference the four eo archives for this release")

    published = dict(
        (name, digest)
        for digest, name in re.findall(r"^([0-9a-f]{64})\s+\*?(\S+)$", checksums, re.MULTILINE)
    )
    if any(published.get(name) != digest for digest, name in pairs):
        raise ValueError("Cask checksums do not match the published release")


if __name__ == "__main__":
    tag, cask_path, checksums_path = sys.argv[1:]
    try:
        verify(tag, Path(cask_path).read_text(), Path(checksums_path).read_text())
    except ValueError as error:
        sys.exit(str(error))
