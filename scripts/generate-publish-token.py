#!/usr/bin/env python3
"""Print a three-word public publishing credential."""

import hashlib
import secrets
import urllib.request


WORDLIST_URL = "https://www.eff.org/files/2016/07/18/eff_large_wordlist.txt"
WORDLIST_SHA256 = "addd35536511597a02fa0a9ff1e5284677b8883b83e986e43f15a3db996b903e"


def main():
    with urllib.request.urlopen(WORDLIST_URL, timeout=15) as response:
        content = response.read()
    if hashlib.sha256(content).hexdigest() != WORDLIST_SHA256:
        raise RuntimeError("EFF wordlist checksum changed")
    words = [line.split()[1] for line in content.decode("ascii").splitlines()]
    words = [word for word in words if word.isalpha() and word.islower()]
    if len(words) != 7772 or len(set(words)) != len(words):
        raise RuntimeError("EFF wordlist has unexpected contents")
    print("-".join(secrets.choice(words) for _ in range(3)))


if __name__ == "__main__":
    main()
