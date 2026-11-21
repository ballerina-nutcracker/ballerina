"""Shared standard-library validation of frozen comparison identities."""

from datetime import date, datetime, time, timezone
import re
from zoneinfo import ZoneInfo

COLOMBO = ZoneInfo("Asia/Colombo")


def validate_manifest(value):
    keys = {
        "version",
        "date",
        "timezone",
        "midnight_utc",
        "cutoff_utc",
        "midnight_local",
        "cutoff_local",
        "default_branch",
        "tip_sha",
        "control_sha",
        "base_sha",
        "head_sha",
        "skipped",
        "run_id",
        "attempt",
        "repository",
    }
    if (
        not isinstance(value, dict)
        or set(value) != keys
        or type(value["version"]) is not int
        or value["version"] != 1
    ):
        raise ValueError("invalid manifest schema/version")
    if not isinstance(value["date"], str) or not re.fullmatch(
        r"\d{4}-\d{2}-\d{2}", value["date"]
    ):
        raise ValueError("invalid comparison date")
    day = date.fromisoformat(value["date"])
    if value["timezone"] != "Asia/Colombo":
        raise ValueError("invalid comparison timezone")
    for name, hour in (("midnight", 0), ("cutoff", 23)):
        instant = datetime.combine(day, time(hour), COLOMBO)
        if (
            value[name + "_local"] != instant.isoformat()
            or value[name + "_utc"] != instant.astimezone(timezone.utc).isoformat()
        ):
            raise ValueError("manifest cutoff/date mismatch")
    for name in ("tip_sha", "control_sha", "base_sha", "head_sha"):
        sha = value[name]
        if name in ("base_sha", "head_sha") and sha is None:
            continue
        if not isinstance(sha, str) or not re.fullmatch(r"[a-f0-9]{40}", sha):
            raise ValueError("invalid manifest SHA")
    if any(
        type(value[name]) is not int or value[name] <= 0
        for name in ("run_id", "attempt")
    ):
        raise ValueError("invalid manifest run/attempt")
    if not isinstance(value["repository"], str) or not re.fullmatch(
        r"[\w.-]+/[\w.-]+", value["repository"], re.ASCII
    ):
        raise ValueError("invalid repository")
    branch = value["default_branch"]
    if (
        not isinstance(branch, str)
        or not branch
        or any(c in branch for c in "\x00\r\n")
    ):
        raise ValueError("invalid default branch")
    skip = value["skipped"]
    if skip not in (None, "unchanged", "insufficient_history"):
        raise ValueError("invalid skip reason")
    base, head = value["base_sha"], value["head_sha"]
    if skip == "unchanged" and (base is None or base != head):
        raise ValueError("invalid unchanged comparison")
    if skip == "insufficient_history" and base is not None and head is not None:
        raise ValueError("invalid insufficient history comparison")
    if skip is None and (base is None or head is None or base == head):
        raise ValueError("invalid measured comparison")
    return value
