#!/usr/bin/env python3
"""Fail if any page loads third-party resources (scripts, styles, images, iframes, forms, fonts).

<a href> links to other sites are plain navigation and are allowed; what is not allowed is the
browser contacting a third party on its own when a page is opened.

    python tools/check_no_external.py dist
"""
import re
import sys
from html.parser import HTMLParser
from pathlib import Path

LOAD_ATTRS = {("img", "src"), ("img", "srcset"), ("script", "src"), ("iframe", "src"), ("frame", "src"),
              ("embed", "src"), ("object", "data"), ("source", "src"), ("source", "srcset"), ("video", "src"),
              ("video", "poster"), ("audio", "src"), ("track", "src"), ("form", "action"), ("input", "src"),
              ("image", "href"), ("use", "href")}
LOAD_LINK_RELS = {"stylesheet", "icon", "apple-touch-icon", "preload", "prefetch", "modulepreload", "manifest", "preconnect", "dns-prefetch"}
EXT = re.compile(r"^(https?:)?//", re.I)
CSS_URL = re.compile(r"url\(\s*['\"]?((?:https?:)?//[^)'\"\s]+)", re.I)
CSS_IMPORT = re.compile(r"@import\s+(?:url\()?\s*['\"]?((?:https?:)?//[^)'\";\s]+)", re.I)


class P(HTMLParser):
    def __init__(self):
        super().__init__()
        self.found, self._style = [], False

    def handle_starttag(self, tag, attrs):
        a = dict(attrs)
        for (t, k) in LOAD_ATTRS:
            if tag == t and a.get(k):
                vals = [x.strip().split()[0] for x in a[k].split(",") if x.strip()] if k == "srcset" else [a[k]]
                self.found += [f"<{tag} {k}> {v}" for v in vals if EXT.match(v)]
        if tag == "link" and a.get("href") and EXT.match(a["href"]):
            rels = set((a.get("rel") or "").lower().split())
            if rels & LOAD_LINK_RELS:
                self.found.append(f"<link rel={a.get('rel')}> {a['href']}")
        if tag == "meta" and (a.get("http-equiv") or "").lower() == "refresh" and re.search(r"url=\s*(https?:)?//", a.get("content", ""), re.I):
            self.found.append("<meta refresh> " + a["content"])
        if a.get("style"):
            self.found += [f"style= {u}" for u in CSS_URL.findall(a["style"])]
        self._style = tag == "style"

    def handle_endtag(self, tag):
        if tag == "style":
            self._style = False

    def handle_data(self, data):
        if self._style:
            self.found += [f"<style> {u}" for u in CSS_URL.findall(data) + CSS_IMPORT.findall(data)]


def main():
    root = Path(sys.argv[1] if len(sys.argv) > 1 else "dist")
    bad = 0
    for f in sorted(root.rglob("*.html")):
        p = P()
        p.feed(f.read_text(encoding="utf8", errors="replace"))
        for item in p.found:
            print(f"{f.relative_to(root)}: {item}")
            bad += 1
    for f in sorted(list(root.rglob("*.css")) + list(root.rglob("*.js"))):
        if "pagefind" in f.parts:
            continue
        t = f.read_text(encoding="utf8", errors="replace")
        hits = CSS_URL.findall(t) + CSS_IMPORT.findall(t) if f.suffix == ".css" else []
        for u in hits:
            print(f"{f.relative_to(root)}: url() {u}")
            bad += 1
    print("OK: no page loads external resources" if not bad else f"{bad} external references")
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
