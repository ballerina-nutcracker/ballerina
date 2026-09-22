#!/usr/bin/env python3
from __future__ import annotations

import re
import sys

ALLOWED_TYPES = (
    "feat",
    "fix",
    "build",
    "chore",
    "ci",
    "docs",
    "style",
    "refactor",
    "perf",
    "test",
    "revert",
)
MAX_SUBJECT_LENGTH = 72
EXPECTED_FORMAT = "<type>(<optional scope>): <description>"

_SUBJECT_PATTERN = re.compile(
    rf"^(?:{'|'.join(ALLOWED_TYPES)})(?:\([^)]+\))?!?: [a-z].*$"
)


def validation_errors(subject: str) -> list[str]:
    errors = []
    if not _SUBJECT_PATTERN.fullmatch(subject):
        errors.append("does not match the Conventional Commits format")
    if len(subject) > MAX_SUBJECT_LENGTH:
        errors.append(
            f"is {len(subject)} characters, exceeding the "
            f"{MAX_SUBJECT_LENGTH}-character limit"
        )
    return errors


def describe_rules() -> str:
    return "\n".join(
        (
            f"Expected: {EXPECTED_FORMAT}",
            f"Allowed types: {', '.join(ALLOWED_TYPES)}",
            "The scope is optional and unrestricted.",
            "A breaking-change marker (!) before the colon is optional.",
            "The description must start with a lowercase letter.",
            f"Maximum length: {MAX_SUBJECT_LENGTH} characters.",
        )
    )


def main(arguments: list[str]) -> int:
    if len(arguments) != 1:
        print(
            "Error: expected exactly one PR title or commit subject argument.\n\n"
            + describe_rules(),
            file=sys.stderr,
        )
        return 2

    subject = arguments[0]
    errors = validation_errors(subject)
    if errors:
        print(
            f'Invalid title or subject "{subject}":\n'
            + "\n".join(f"- {error}" for error in errors)
            + "\n\n"
            + describe_rules(),
            file=sys.stderr,
        )
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
