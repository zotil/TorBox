#!/usr/bin/env python3
"""Generate dist/ from site/ by substituting the tokens from config.json.

    python tools/build.py [--config config.json] [--site site] [--dist dist]

Tokens:
  @@SITE_BASE_URL@@       full public URL of the site, with trailing "/" (canonical, sitemap.xml)
  @@SITE_PATH@@           only the path of SITE_BASE_URL ("/TorBox/" or "/"); used by 404.html
  @@DOWNLOAD_BASE_URL@@   base URL from which the images (.img.xz, .gz) are downloaded
"""
import argparse
import json
import shutil
import sys
import urllib.parse
from pathlib import Path

TEXT_SUFFIXES = {".html", ".js", ".css", ".xml", ".txt", ".json"}


def main():
    root = Path(__file__).resolve().parent.parent
    ap = argparse.ArgumentParser()
    ap.add_argument("--config", default=str(root / "config.json"))
    ap.add_argument("--site", default=str(root / "site"))
    ap.add_argument("--dist", default=str(root / "dist"))
    args = ap.parse_args()

    cfg = json.load(open(args.config, encoding="utf8"))
    site_url = cfg["SITE_BASE_URL"].rstrip("/") + "/"
    dl_url = cfg["DOWNLOAD_BASE_URL"].rstrip("/") + "/"
    p = urllib.parse.urlsplit(site_url)
    if p.scheme not in ("http", "https") or not p.netloc:
        sys.exit("SITE_BASE_URL is not a valid URL: " + site_url)
    tokens = {"@@SITE_BASE_URL@@": site_url, "@@SITE_PATH@@": p.path or "/", "@@DOWNLOAD_BASE_URL@@": dl_url}
    if ".invalid" in dl_url or "example." in dl_url:
        print("WARNING: DOWNLOAD_BASE_URL is still the placeholder; download links will not work:", dl_url)

    site, dist = Path(args.site), Path(args.dist)
    if dist.exists():
        shutil.rmtree(dist)
    shutil.copytree(site, dist)

    n = 0
    for f in dist.rglob("*"):
        if f.is_file() and f.suffix in TEXT_SUFFIXES:
            t = f.read_text(encoding="utf8")
            if "@@" in t:
                for k, v in tokens.items():
                    t = t.replace(k, v)
                f.write_text(t, encoding="utf8")
                n += 1
            if "@@" in t:
                import re
                left = set(re.findall(r"@@[A-Z_]+@@", t))
                if left:
                    sys.exit(f"{f}: unresolved tokens {left}")

    urls = []
    for f in sorted(dist.rglob("index.html")):
        d = f.parent.relative_to(dist).as_posix()
        if d in ("search",) or "/page/" in "/" + d + "/":
            continue
        urls.append(site_url + (urllib.parse.quote(d) + "/" if d != "." else ""))
    (dist / "sitemap.xml").write_text(
        '<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n'
        + "".join(f"  <url><loc>{u}</loc></url>\n" for u in urls) + "</urlset>\n", encoding="utf8")
    (dist / "robots.txt").write_text(f"User-agent: *\nAllow: /\n\nSitemap: {site_url}sitemap.xml\n", encoding="utf8")
    print(f"dist ready: {n} files with tokens, {len(urls)} URLs in sitemap, base {site_url}")


if __name__ == "__main__":
    main()
