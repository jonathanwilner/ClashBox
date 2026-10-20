#!/usr/bin/env python3
"""Preserve the ARM64 runtime TLS helper register contract under TLSDESC."""
import sys
from pathlib import Path
p = Path(sys.argv[1]) / "src/runtime/tls_arm64.s"
s = p.read_text()
old = "\tMOVD\tR1, R10\n\tMOVD\tR30, R11\n\tMOVD\truntime·tls_g(SB), R0\n\tMOVD\tR10, R1\n\tMOVD\tR11, R30"
first_repair = "\tMOVD.W\tR1, -16(RSP)\n\tMOVD\tR30, 8(RSP)\n\tMOVD\truntime·tls_g(SB), R0\n\tMOVD\t8(RSP), R30\n\tMOVD.P\t16(RSP), R1"
# Go ARM64 saves the caller frame pointer at SP-8. Leave that slot untouched.
# A TLSDESC resolver may push its own registers below the adjusted SP.
new = first_repair.replace("-16(RSP)", "-32(RSP)").replace("16(RSP), R1", "32(RSP), R1")
if s.count(old) == 2:
    s = s.replace(old, new)
elif s.count(first_repair) == 2:
    s = s.replace(first_repair, new)
elif s.count(new) != 2:
    raise SystemExit("Unexpected TLS helper source; refusing partial patch")
s = s.replace("so we save R1 and LR to scratch registers R10 and R11.", "so we save R1 and LR on the stack, preserving all other registers.")
p.write_text(s)
