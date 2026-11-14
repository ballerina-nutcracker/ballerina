"""Plain display text in Markdown reports; no numerical dependencies."""

import html
import re


def display(value):
    text = str(value).replace("@", "@\u200b").replace("\r", " ").replace("\n", " ")
    text = re.sub(r"([\\`*_{}\[\]()#+!|])", r"\\\1", text)
    return html.escape(text, quote=False)


def short_display(value):
    text = str(value)
    return display(text if len(text) <= 180 else text[:177] + "...")
