#!/bin/sh
# Integration checks for yup-yes, run inside a Debian (GNU coreutils) container.
#
# yes is an infinite stream, so every comparison is capped with `head -n N` on
# BOTH sides to terminate. parity holds for the no-operand and single-operand
# forms; the multi-operand join and the -n/--count limit are yup-yes extensions
# with no GNU equivalent, asserted exactly (see cmd-yes COMPATIBILITY.md).
#
# parity CASE  — `yup-yes CASE | head -n N` must equal `yes CASE | head -n N`.
# assert WANT  — yup-yes must produce WANT exactly (documented divergences).
set -eu

fails=0
cap=3

parity() {
	ours=$(yup-yes "$@" 2>/dev/null | head -n "$cap" || true)
	gnu=$(yes "$@" 2>/dev/null | head -n "$cap" || true)
	if [ "$ours" = "$gnu" ]; then
		printf 'ok    parity  yes %s | head -n %s\n' "$*" "$cap"
	else
		printf 'FAIL  parity  yes %s | head -n %s\n        gnu:  %s\n        ours: %s\n' "$*" "$cap" "$gnu" "$ours"
		fails=$((fails + 1))
	fi
}

assert() {
	want=$1
	shift
	got=$(yup-yes "$@" 2>/dev/null || true)
	if [ "$got" = "$want" ]; then
		printf 'ok    assert  yes %s\n' "$*"
	else
		printf 'FAIL  assert  yes %s\n        want: %s\n        got:  %s\n' "$*" "$want" "$got"
		fails=$((fails + 1))
	fi
}

# Default operand: repeats "y" (matches GNU when both are capped with head).
parity

# Single STRING operand: repeats that string verbatim (matches GNU).
parity hello
parity foo

# Documented divergence: an explicit empty operand collapses to the default
# "y" (cmd-yes treats an empty text as "unset"), whereas GNU `yes ""` repeats a
# blank line. Asserted with the count extension to keep the output finite.
assert "$(printf 'y\ny')" -n 2 ""

# -n / --count: a yup-yes extension (GNU yes has no count flag). It bounds the
# stream itself, so no head cap is needed; assert the exact finite output.
assert "$(printf 'y\ny\ny')" -n 3
assert "$(printf 'ok\nok')" --count 2 ok

# Multiple operands: yup-yes joins them with spaces; GNU yes uses only the
# first. A documented divergence, so assert the joined form (capped to 2 lines
# via -n to keep the output finite and deterministic).
assert "$(printf 'one two three\none two three')" -n 2 one two three

if [ "$fails" -ne 0 ]; then
	printf '\n%s check(s) failed\n' "$fails"
	exit 1
fi
printf '\nall checks passed\n'
