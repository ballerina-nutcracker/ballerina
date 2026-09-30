import email.message
import ssl
import tempfile
import unittest
import urllib.error
from pathlib import Path
from unittest import mock

import check_doc_links


class CheckDocLinksTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)

    def write(self, name, content):
        path = Path(self.tmp.name) / name
        path.write_text(content, encoding="utf-8")
        return path

    def targets(self, content):
        return [link.target for link in check_doc_links.extract_links(self.write("doc.md", content))]

    def test_link_destination_with_balanced_parentheses(self):
        self.assertEqual(self.targets("[doc](guide(v2).md)\n"), ["guide(v2).md"])

    def test_angle_bracket_link_destination(self):
        self.assertEqual(self.targets("[doc](<my guide.md>)\n"), ["my guide.md"])

    def test_autolink_is_extracted(self):
        self.assertEqual(self.targets("See <https://example.com/a>.\n"), ["https://example.com/a"])

    def test_image_nested_in_link_is_extracted(self):
        self.assertEqual(
            self.targets("[![Build](https://example.com/badge.svg)](https://example.com/ci)\n"),
            ["https://example.com/badge.svg", "https://example.com/ci"],
        )

    def test_links_in_code_are_ignored(self):
        self.assertEqual(self.targets("`[a](x.md)` and `<https://example.com>`\n"), [])

    def test_shorter_fence_does_not_close_longer_fence(self):
        content = "````\n```\n[example](missing.md)\n```\n````\n[real](real.md)\n"
        self.assertEqual(self.targets(content), ["real.md"])

    def test_fence_with_trailing_text_does_not_close_fence(self):
        content = "```\n``` not a close\n[example](missing.md)\n```\n[real](real.md)\n"
        self.assertEqual(self.targets(content), ["real.md"])

    def test_setext_headings_are_anchors(self):
        path = self.write("doc.md", "Overview\n========\n\nDetails here\n---\n")
        self.assertTrue({"overview", "details-here"} <= check_doc_links.markdown_anchors(path))

    def test_thematic_break_after_blank_line_is_not_a_heading(self):
        path = self.write("doc.md", "Text\n\n---\n")
        self.assertEqual(check_doc_links.markdown_anchors(path), set())

    def test_cloudflare_challenge_is_treated_as_reachable(self):
        headers = email.message.Message()
        headers["cf-mitigated"] = "challenge"
        challenge = urllib.error.HTTPError("https://example.com", 403, "Forbidden", headers, None)
        with mock.patch("urllib.request.urlopen", side_effect=challenge):
            self.assertIsNone(check_doc_links.fetch_page("https://example.com").error)

    def test_plain_forbidden_is_an_error(self):
        forbidden = urllib.error.HTTPError("https://example.com", 403, "Forbidden", email.message.Message(), None)
        with mock.patch("urllib.request.urlopen", side_effect=forbidden):
            self.assertEqual(check_doc_links.fetch_page("https://example.com").error, "HTTP 403")

    def test_request_errors_are_reported(self):
        failures = [UnicodeEncodeError("ascii", "é", 0, 1, "ordinal not in range"), ssl.SSLError("bad record")]
        for failure in failures:
            with (
                self.subTest(failure=type(failure).__name__),
                mock.patch("urllib.request.urlopen", side_effect=failure),
                mock.patch("time.sleep"),
            ):
                error = check_doc_links.fetch_page("https://example.com/café").error
                self.assertTrue(error.startswith("request failed"))


if __name__ == "__main__":
    unittest.main()
