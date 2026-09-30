#!/usr/bin/env python3
"""Validate links in the repository's Markdown files.

Relative links must point to an existing file or directory, and a fragment on a
link to a Markdown file must match one of its headings. External links must
respond successfully, and a fragment on an external HTML page must match an
element id or name on that page.
"""
from __future__ import annotations

import argparse
import html.parser
import http.client
import re
import subprocess
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
USER_AGENT = "Mozilla/5.0 (compatible; ballerina-doc-link-check)"
FETCH_ATTEMPTS = 3
FETCH_TIMEOUT_SECONDS = 30

FENCE_RE = re.compile(r"^ {0,3}(`{3,}|~{3,})(.*)$")
INLINE_CODE_RE = re.compile(r"(`+).+?\1")
INLINE_LINK_RE = re.compile(
    r"!?\[((?:[^\[\]]|\[[^\]]*\])*)\]\(\s*"
    r"(?:<([^<>\n]*)>|((?:[^()\s\\]|\\.|\([^()\s]*\))+))"
    r"(?:\s+(?:\"[^\"]*\"|'[^']*'|\([^)]*\)))?\s*\)"
)
AUTOLINK_RE = re.compile(r"<(https?://[^\s<>]+)>")
BACKSLASH_ESCAPE_RE = re.compile(r"\\(.)")
REFERENCE_DEF_RE = re.compile(r"^\s{0,3}\[[^\]]+\]:\s*<?(\S+?)>?(?:\s+.*)?$")
HEADING_RE = re.compile(r"^\s{0,3}(#{1,6})\s+(.*?)\s*#*\s*$")
SETEXT_UNDERLINE_RE = re.compile(r"^ {0,3}(=+|-+)\s*$")
HTML_ANCHOR_RE = re.compile(r"<a\s[^>]*(?:name|id)=\"([^\"]+)\"", re.IGNORECASE)
GITHUB_LINE_FRAGMENT_RE = re.compile(r"^L\d+(C\d+)?(-L\d+(C\d+)?)?$")


@dataclass(frozen=True)
class Link:
    source: Path
    line: int
    target: str


@dataclass
class Page:
    error: str | None
    anchors: set[str]


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("files", nargs="*", type=Path, help="Markdown files to check (default: all tracked *.md files)")
    parser.add_argument("--offline", action="store_true", help="skip external links")
    args = parser.parse_args()

    files = [path.resolve() for path in args.files] or tracked_markdown_files()
    links = [link for path in files for link in extract_links(path)]
    errors = check_local_links(links)
    if not args.offline:
        errors += check_external_links(links)

    for error in sorted(errors):
        print(error, file=sys.stderr)
    print(f"Checked {len(links)} links in {len(files)} files: {len(errors)} broken")
    return 1 if errors else 0


def tracked_markdown_files() -> list[Path]:
    output = subprocess.run(
        ["git", "ls-files", "-z", "*.md"], cwd=REPO_ROOT, check=True, capture_output=True, text=True
    ).stdout
    return [REPO_ROOT / name for name in output.split("\0") if name]


def extract_links(path: Path) -> list[Link]:
    links = []
    for number, line in markdown_lines(path):
        text = INLINE_CODE_RE.sub("", line)
        targets = inline_link_targets(text)
        targets += AUTOLINK_RE.findall(text)
        definition = REFERENCE_DEF_RE.match(text)
        if definition:
            targets.append(definition.group(1))
        links.extend(Link(path, number, target) for target in targets)
    return links


def inline_link_targets(text: str) -> list[str]:
    targets = []
    for match in INLINE_LINK_RE.finditer(text):
        targets += inline_link_targets(match.group(1))
        targets.append(inline_link_target(match))
    return targets


def inline_link_target(match: re.Match[str]) -> str:
    if match.group(2) is not None:
        return match.group(2)
    return BACKSLASH_ESCAPE_RE.sub(r"\1", match.group(3))


def markdown_lines(path: Path) -> list[tuple[int, str]]:
    lines = []
    fence = None
    for number, line in enumerate(path.read_text(encoding="utf-8").splitlines(), start=1):
        marker = FENCE_RE.match(line)
        if fence is None and marker:
            fence = marker.group(1)
        elif fence is not None and marker and closes_fence(fence, marker):
            fence = None
        elif fence is None:
            lines.append((number, line))
    return lines


def closes_fence(fence: str, marker: re.Match[str]) -> bool:
    closing = marker.group(1)
    return closing[0] == fence[0] and len(closing) >= len(fence) and not marker.group(2).strip()


def check_local_links(links: list[Link]) -> list[str]:
    anchor_cache: dict[Path, set[str]] = {}
    errors = []
    for link in links:
        if is_external(link.target) or link.target.startswith("mailto:"):
            continue
        path_part, _, fragment = link.target.partition("#")
        target = resolve_local_path(link.source, urllib.parse.unquote(path_part))
        if not target.exists():
            errors.append(describe(link, "file not found"))
            continue
        if not fragment or target.suffix.lower() != ".md":
            continue
        if target not in anchor_cache:
            anchor_cache[target] = markdown_anchors(target)
        if urllib.parse.unquote(fragment).lower() not in anchor_cache[target]:
            errors.append(describe(link, f"no heading or anchor '{fragment}' in {relative(target)}"))
    return errors


def resolve_local_path(source: Path, path_part: str) -> Path:
    if not path_part:
        return source
    if path_part.startswith("/"):
        return REPO_ROOT / path_part.lstrip("/")
    return (source.parent / path_part).resolve()


def markdown_anchors(path: Path) -> set[str]:
    anchors = set()
    slug_counts: dict[str, int] = {}
    previous: tuple[int, str] | None = None
    for number, line in markdown_lines(path):
        anchors.update(anchor.lower() for anchor in HTML_ANCHOR_RE.findall(line))
        heading_text = heading_of(line, previous if previous and previous[0] == number - 1 else None)
        previous = (number, line)
        if heading_text is None:
            continue
        slug = github_slug(heading_text)
        count = slug_counts.get(slug, 0)
        slug_counts[slug] = count + 1
        anchors.add(slug if count == 0 else f"{slug}-{count}")
    return anchors


def heading_of(line: str, previous: tuple[int, str] | None) -> str | None:
    atx = HEADING_RE.match(line)
    if atx:
        return atx.group(2)
    if previous and SETEXT_UNDERLINE_RE.match(line) and is_setext_text(previous[1]):
        return previous[1].strip()
    return None


def is_setext_text(line: str) -> bool:
    return bool(line.strip()) and not HEADING_RE.match(line) and not SETEXT_UNDERLINE_RE.match(line)


def github_slug(heading: str) -> str:
    text = re.sub(r"!?\[([^\]]*)\]\([^)]*\)", r"\1", heading)
    text = re.sub(r"<[^>]+>", "", text)
    text = re.sub(r"[^\w\- ]", "", text.strip().lower())
    return text.replace(" ", "-")


def check_external_links(links: list[Link]) -> list[str]:
    external = [link for link in links if is_external(link.target)]
    urls = sorted({strip_fragment(link.target) for link in external})
    with ThreadPoolExecutor(max_workers=8) as pool:
        pages = dict(zip(urls, pool.map(fetch_page, urls)))

    errors = []
    for link in external:
        url, _, fragment = link.target.partition("#")
        page = pages[url]
        if page.error:
            errors.append(describe(link, page.error))
        elif fragment and not fragment_exists(url, urllib.parse.unquote(fragment), page):
            errors.append(describe(link, f"no element with id or name '{fragment}' on the page"))
    return errors


def fetch_page(url: str) -> Page:
    error = "not fetched"
    for attempt in range(FETCH_ATTEMPTS):
        if attempt:
            time.sleep(2**attempt)
        try:
            request = urllib.request.Request(url, headers={"User-Agent": USER_AGENT})
            with urllib.request.urlopen(request, timeout=FETCH_TIMEOUT_SECONDS) as response:
                content_type = response.headers.get("Content-Type", "")
                body = response.read().decode("utf-8", errors="replace") if "html" in content_type else ""
                return Page(None, html_anchors(body))
        except urllib.error.HTTPError as err:
            if is_bot_challenge(err):
                return Page(None, set())
            error = f"HTTP {err.code}"
            if err.code < 500 and err.code != 429:
                break
        except (OSError, ValueError, http.client.HTTPException) as err:
            error = f"request failed: {getattr(err, 'reason', err)}"
    return Page(error, set())


def is_bot_challenge(err: urllib.error.HTTPError) -> bool:
    return err.code == 403 and err.headers.get("cf-mitigated") == "challenge"


class AnchorCollector(html.parser.HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.anchors: set[str] = set()

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        for name, value in attrs:
            if name in ("id", "name") and value:
                self.anchors.add(value)


def html_anchors(body: str) -> set[str]:
    collector = AnchorCollector()
    collector.feed(body)
    return collector.anchors


def fragment_exists(url: str, fragment: str, page: Page) -> bool:
    if fragment in page.anchors:
        return True
    if urllib.parse.urlparse(url).hostname == "github.com":
        return GITHUB_LINE_FRAGMENT_RE.match(fragment) is not None or f"user-content-{fragment}" in page.anchors
    return False


def is_external(target: str) -> bool:
    return target.startswith(("http://", "https://"))


def strip_fragment(target: str) -> str:
    return target.partition("#")[0]


def describe(link: Link, problem: str) -> str:
    return f"{relative(link.source)}:{link.line}: {link.target}: {problem}"


def relative(path: Path) -> str:
    try:
        return str(path.relative_to(REPO_ROOT))
    except ValueError:
        return str(path)


if __name__ == "__main__":
    sys.exit(main())
