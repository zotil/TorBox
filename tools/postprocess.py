#!/usr/bin/env python3
"""Convert a WordPress mirror (wget --mirror) into the clean static TorBox site.

Usage:
    python tools/postprocess.py --mirror ../work/mirror --webroot ../work/wp/html --out site

Inputs:
  --mirror   output of `wget --mirror` against the local WordPress (http://127.0.0.1:8080)
  --webroot  webroot of the local WordPress; only used to copy files that the
             HTML references but wget did not download (e.g. /data/*.zip, theme banner)
  tools/data/wp-ids.json   ID -> slug / attachment file map, exported from the database
  tools/data/external/     remote resources the content used to embed, now self-hosted

Tokens that tools/build.py later substitutes from config.json:
  @@DOWNLOAD_BASE_URL@@   download base for the images (.img.xz, .gz)
  @@SITE_BASE_URL@@       public URL of the site (canonical, sitemap)
  @@SITE_PATH@@           base path of the site (only in 404.html)
"""
import argparse
import html as htmllib
import json
import os
import re
import shutil
import sys
import urllib.parse
from pathlib import Path

from bs4 import BeautifulSoup, Comment

HERE = Path(__file__).resolve().parent
LOCAL_HOSTS = {"127.0.0.1:8080", "www.torbox.ch", "torbox.ch"}
HOST_RE = re.compile(r"https?://(?:127\.0\.0\.1:8080|www\.torbox\.ch|torbox\.ch)/([^)\"'\s<>]*)")

DL_TOKEN = "@@DOWNLOAD_BASE_URL@@"
SITE_TOKEN = "@@SITE_BASE_URL@@"
PATH_TOKEN = "@@SITE_PATH@@"

ORIG_REPO = "https://github.com/radio24/TorBox"      # the site only links to the repository under its original name (GitHub redirects)
DISCUSSIONS = ORIG_REPO + "/discussions"

# WordPress slugs with non-ASCII characters -> stable slug
SLUG_OVERRIDE = {
    "torbox-tft-based-on-the-pitft-3-5″-resistive-touch-320x480-from-adafruit": "torbox-tft",
}
HOME_ID = 2
# IDs that no longer exist as a page but whose old links must land on the home page
REDIRECT_TO_HOME = set()

EXTERNAL_IMAGES = {
    "https://user-images.githubusercontent.com/46644350/151696060-efeb927b-aea8-405c-92ba-a9c4c3de5ff9.png":
        "assets/external/github-151696060.png",
}

SH_STYLES = ["wp-content/plugins/syntaxhighlighter/syntaxhighlighter3/styles/shCore.css",
             "wp-content/plugins/syntaxhighlighter/syntaxhighlighter3/styles/shThemeDefault.css"]

CSP = ("default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; "
       "script-src 'self' 'unsafe-inline' 'wasm-unsafe-eval'; font-src 'self' data:; connect-src 'self'; "
       "frame-src 'none'; object-src 'none'; form-action 'self'; base-uri 'self'")


class Ctx:
    def __init__(self, args):
        self.out = Path(args.out)
        self.webroot = Path(args.webroot)
        ids = json.load(open(HERE / "data" / "wp-ids.json", encoding="utf8"))
        self.content = ids["content"]
        self.attach = ids["attachments"]
        self.cats = ids["categories"]
        self.refs = set()          # local paths referenced (to verify they exist)
        self.dead = []             # (page, url) without a target
        self.unwrapped = 0
        self.retired = []

    def slug_path(self, cid):
        cid = int(cid)
        if cid == HOME_ID or cid in REDIRECT_TO_HOME:
            return ""
        c = self.content.get(str(cid))
        if not c or c["status"] != "publish":
            return None
        return SLUG_OVERRIDE.get(c["slug"], c["slug"]) + "/"


def rel_to(page_dir, target):
    """Relative URL from the page directory to the target (a path from the site root)."""
    is_dir = target == "" or target.endswith("/")
    r = os.path.relpath(target.rstrip("/") or ".", page_dir or ".")
    r = urllib.parse.quote(r.replace(os.sep, "/"), safe="/%@:+,~()'!*=&;$")
    if is_dir:
        r = "./" if r == "." else r.rstrip("/") + "/"
    return r


TAGS_URL = ORIG_REPO + "/tags"
CURRENT_IMAGES = {"torbox-20260322-v055.img.xz", "torbox-mini-20260322-v055.img.xz"}


def download_target(ctx, name):
    """Download policy: only the two 2026-03-22 images link to the download server; earlier images are
    dropped (None => the link is left as plain text); menu zips that are not in the site
    point to the repository tags."""
    low = name.lower()
    if low.endswith(".img.xz"):
        return DL_TOKEN + urllib.parse.quote(name) if low in CURRENT_IMAGES else None
    if low.endswith(".gz"):
        return None
    if (ctx.out / "data" / name).exists() or (ctx.webroot / "data" / name).exists():
        return "data/" + name
    if low.endswith(".zip"):
        return TAGS_URL
    return None


def rewrite_url(ctx, u, page_dir, page_name):
    """Return the rewritten URL, or None if the target no longer exists (the link is unwrapped)."""
    if not u:
        return u
    u0 = u.strip()
    if u0.startswith("//www.torbox.ch") or u0.startswith("//torbox.ch"):
        u0 = "https:" + u0.rstrip("\u201d\u2019")             # scheme-less link with a typographic quote attached
    if u0 in EXTERNAL_IMAGES:
        return rel_to(page_dir, EXTERNAL_IMAGES[u0])
    p = urllib.parse.urlsplit(u0)
    if p.scheme not in ("http", "https") or p.netloc.lower() not in LOCAL_HOSTS:
        return u
    path = urllib.parse.unquote(p.path)
    q = urllib.parse.parse_qs(p.query)
    frag = ("#" + p.fragment) if p.fragment else ""
    target = None
    if "attachment_id" in q:
        f = ctx.attach.get(q["attachment_id"][0])
        target = ("wp-content/uploads/" + f) if f else None
    elif "page_id" in q or "p" in q:
        target = ctx.slug_path((q.get("page_id") or q["p"])[0])
    elif path.startswith("/data/"):
        name = path[len("/data/"):]
        t = download_target(ctx, name)
        if t is None:
            ctx.retired.append((page_name, name))
            return None
        if t.startswith(DL_TOKEN) or t.startswith("https://"):
            return t + frag
        target = t
    elif "/author/" in path:
        return None
    else:
        target = path.lstrip("/")
        target = SLUG_OVERRIDE.get(target.rstrip("/"), target.rstrip("/")) + ("/" if target.endswith("/") else "")
        if target.rstrip("/") in SLUG_OVERRIDE.values() and not target.endswith("/"):
            target += "/"
    if target is None:
        ctx.dead.append((page_name, u0))
        return None
    ctx.refs.add(target)
    return rel_to(page_dir, target) + frag


def rewrite_css_text(text, base_dir):
    """Replace local absolute URLs inside CSS/JS/style attributes with relative paths."""
    def sub(m):
        path = urllib.parse.unquote(m.group(1)).split("#")[0].split("?")[0]
        return rel_to(base_dir, path) if path else "./"
    return HOST_RE.sub(sub, text)


# ---------------------------------------------------------------- new HTML fragments

def comments_notice():
    return f"""
<p class="comments-notice">Comments on this site are closed. The comments below were published while the
previous WordPress site was running and are kept as an archive. To ask a question or discuss TorBox, please use
<a href="{DISCUSSIONS}" rel="noopener noreferrer">GitHub Discussions (radio24/TorBox)</a>.</p>"""


def frag(s):
    return BeautifulSoup(s, "html.parser")


# ---------------------------------------------------------------- per-page processing

EXT_ATTRS = ("src", "href", "srcset", "data-src", "poster", "action")


def embed_placeholder(src, title):
    p = urllib.parse.urlsplit(src)
    host = p.netloc.lower()
    if "youtube.com" in host or "youtube-nocookie.com" in host:
        vid = p.path.rstrip("/").split("/")[-1]
        url, label = f"https://www.youtube.com/watch?v={vid}", "YouTube"
    elif "livestream.com" in host:
        url, label = f"https://livestream.com{p.path.rsplit('/player', 1)[0]}", "Livestream"
    else:
        url, label = src, host
    t = htmllib.escape(title) + " \u2013 " if title else ""
    return (f'<p class="embed-placeholder">Embedded video not loaded (to avoid contacting third parties). '
            f'{t}<a href="{htmllib.escape(url)}" rel="noopener noreferrer">Watch it on {label}</a> '
            f'(external site; opening the link contacts {label}).</p>')


def process_page(ctx, path):
    rel = path.relative_to(ctx.out)
    page_dir = str(rel.parent) if str(rel.parent) != "." else ""
    page_name = str(rel)
    pre = rel_to(page_dir, "")                      # prefix towards the root: "./" or "../../"
    pre = "" if pre == "./" else pre
    raw = path.read_text(encoding="utf8", errors="surrogateescape")
    soup = BeautifulSoup(raw, "lxml")
    head, body = soup.head, soup.body
    classes = body.get("class", [])
    is_content = "single" in classes or "page" in classes
    is_home = page_dir == ""

    # --- HTML comments
    for c in soup.find_all(string=lambda s: isinstance(s, Comment)):
        c.extract()

    # --- WordPress fingerprints in <head>
    for sel in ['meta[name="generator"]', 'meta[name="robots"]', 'link[rel="profile"]', 'link[rel="pingback"]',
                'link[rel="EditURI"]', 'link[rel="wlwmanifest"]', 'link[rel="shortlink"]', 'link[rel="canonical"]',
                'link[rel="https://api.w.org/"]', 'link[rel="dns-prefetch"]', 'link[rel="preconnect"]',
                'link[rel="alternate"]', "style#wp-emoji-styles-inline-css", "script#wp-emoji-settings",
                "script#comment-reply-js", 'script[src*="wp-emoji"]']:
        for el in soup.select(sel):
            el.decompose()
    for s in soup.find_all("script"):
        if s.get("src") and urllib.parse.urlsplit(s["src"]).netloc.lower() not in LOCAL_HOSTS | {""}:
            s.decompose()                                # twitter widgets, etc.
    for s in soup.find_all("script", string=True):
        if "wp-emoji" in s.string or "_wpemojiSettings" in s.string:
            s.decompose()
    for s in soup.find_all("script", type="speculationrules"):
        s.decompose()
    for s in soup.find_all("script", string=lambda x: x and "corecssurl" in x):
        # SyntaxHighlighter injected its stylesheets via JS with absolute paths: replaced by static <link> tags
        s.string = s.string[s.string.index("SyntaxHighlighter.config"):] if "SyntaxHighlighter.config" in s.string else ""
        for css in SH_STYLES:
            head.append(frag(f'<link rel="stylesheet" href="@@PRE@@{css}">'))
            ctx.refs.add(css)

    # --- author links and links without a target
    for a in soup.select('a[href*="/author/"]'):
        a.unwrap()
    for a in soup.select(".comment-author a"):           # commenters' websites: keep only the name, as text
        a.unwrap()

    # --- emails: malformed links and old addresses
    for a in soup.find_all("a", href=True):
        h = a["href"].strip()
        if h.startswith("emailt:"):
            a["href"] = "mailto:" + h.split(":", 1)[1].strip().replace(" ", "")
        elif h == "http://anonym@torbox.ch":
            a["href"] = "mailto:anonym@torbox.ch"

    # --- iframes -> link
    for fr in soup.find_all("iframe"):
        ph = frag(embed_placeholder(fr.get("src", ""), fr.get("title")))
        fr.replace_with(ph)
    for tag in soup.find_all(["object", "embed"]):
        tag.decompose()
    for bq in soup.select("blockquote.twitter-tweet"):
        bq["class"] = bq.get("class", []) + ["static-tweet"]

    # --- comments: static archive only
    com = soup.select_one("#comments")
    if com:
        for sel in ["#respond", ".comment-respond", ".reply", ".comment-navigation", ".no-comments", ".comment-reply-title"]:
            for el in com.select(sel):
                el.decompose()
        lst = com.select_one("ol.comment-list")
        title = com.select_one(".comments-title")
        if lst:
            n = len(lst.select("li.comment"))
            if title:
                title.string = f"Archived comments ({n})"
                title["id"] = "comments-title"
            title.insert_after(frag(comments_notice()))
            for sp in lst.select(".says"):
                sp.decompose()
        else:
            com.clear()
            com.append(frag(comments_notice()))

    # --- "Leave a comment" (there is no form any more)
    for a in soup.select('a[href$="#respond"]'):
        holder = a.find_parent("span", class_="comments-link")
        (holder or a).decompose()

    # --- search
    sw = soup.select_one("section.widget_search")
    if sw:
        new = frag('<section id="search-widget" class="widget widget_search"><h2 class="widget-title">Search</h2>'
                   '<div id="search" class="pagefind-host"></div>'
                   '<p class="search-hint"><a href="@@PRE@@search/">Full-page search</a></p></section>')
        sw.replace_with(new)
    # --- footer: no "Proudly powered by WordPress"
    imp = soup.select_one(".site-info a.imprint")
    if imp:
        imp.decompose()                               # "Proudly powered by WordPress": the fingerprint is removed, no new text

    # --- header: search link for mobile
    hm = soup.select_one(".site-header-main")
    if hm:
        hm.append(frag('<a class="search-jump" href="@@PRE@@search/">Search</a>'))

    # --- Pagefind: index only pages and posts
    main = soup.select_one("main#main")
    if main is not None:
        if is_content:
            main["data-pagefind-body"] = ""
        else:
            main["data-pagefind-ignore"] = ""

    # --- new meta tags
    head.insert(1, frag(f'<meta http-equiv="Content-Security-Policy" content="{CSP}">'))
    head.insert(2, frag('<meta name="referrer" content="no-referrer">'))
    c = "" if is_home else (page_dir.replace(os.sep, "/") + "/" if path.name == "index.html" else page_dir)
    if path.name == "index.html":
        head.append(frag(f'<link rel="canonical" href="{SITE_TOKEN}{urllib.parse.quote(c)}">'))
    head.append(frag('<link rel="stylesheet" href="@@PRE@@assets/torbox-static.css">'))
    head.append(frag('<link rel="stylesheet" href="@@PRE@@pagefind/pagefind-ui.css">'))
    body.append(frag('<script src="@@PRE@@pagefind/pagefind-ui.js"></script>'))
    body.append(frag('<script src="@@PRE@@assets/torbox-search.js"></script>'))

    # --- URL rewriting
    for el in soup.find_all(True):
        for attr, v in list(el.attrs.items()):
            if attr in ("srcset", "style") or not isinstance(v, str) or not (v.startswith("http") or v.startswith("//")):
                continue
            nv = rewrite_url(ctx, v, page_dir, page_name)
            if nv is None:
                if attr == "href" and el.name == "a":
                    el.unwrap()
                    ctx.unwrapped += 1
                    break
                del el[attr]
            elif nv != v:
                el[attr] = nv
        if el.parent is None:
            continue
        v = el.get("srcset")
        if isinstance(v, str) and v:
            parts = []
            for item in v.split(","):
                bits = item.strip().split()
                if not bits:
                    continue
                nu = rewrite_url(ctx, bits[0], page_dir, page_name)
                if nu:
                    parts.append(" ".join([nu] + bits[1:]))
            el["srcset"] = ", ".join(parts)
        if el.get("style") and "http" in el["style"]:
            el["style"] = rewrite_css_text(el["style"], page_dir)
    for st in soup.find_all("style"):
        if st.string and "http" in st.string:
            st.string = rewrite_css_text(st.string, page_dir)
    for sc in soup.find_all("script"):
        if sc.string and HOST_RE.search(sc.string):
            sc.string = HOST_RE.sub(lambda m: "/" + m.group(1), sc.string)

    for t in soup.find_all(string=lambda x: "127.0.0.1:8080" in x and x.parent.name not in ("script", "style")):
        t.replace_with(str(t).replace("127.0.0.1:8080", "www.torbox.ch"))   # visible text of old links
    out = str(soup)
    out = out.replace("@@PRE@@", pre)
    path.write_text(out, encoding="utf8", errors="surrogateescape")


# ---------------------------------------------------------------- generated pages and redirect map

CAT_PARENT = {"10": "9", "11": "9", "12": "9", "13": "9", "15": "9"}      # wp_term_taxonomy.parent


def build_redirect_map(ctx):
    out = ctx.out
    m = {"c": {}, "a": {}, "k": {}}
    for cid, c in ctx.content.items():
        if c["status"] != "publish" and int(cid) not in REDIRECT_TO_HOME:
            continue
        t = ctx.slug_path(cid)
        if t is None:
            continue
        if t == "" or (out / t / "index.html").exists():
            m["c"][cid] = t
    for aid, f in ctx.attach.items():
        if (out / "wp-content" / "uploads" / f).exists():
            m["a"][aid] = "wp-content/uploads/" + f
    for kid, slug in ctx.cats.items():
        parts = [slug]
        if kid in CAT_PARENT:
            parts.insert(0, ctx.cats[CAT_PARENT[kid]])
        t = "category/" + "/".join(parts) + "/"
        if (out / t / "index.html").exists():
            m["k"][kid] = t
    return m


REDIRECT_JS = r"""<script>
/* Redirects for old WordPress links (/?page_id=N, /?p=N, /?attachment_id=N, /?cat=N, /?m=YYYYMM, /?s=text).
   Map generated from the database by tools/postprocess.py. */
(function () {
	var s = location.search;
	if (s.length < 2) { return; }
	var M = %s;
	var q = {};
	s.slice(1).split('&').forEach(function (kv) {
		if (!kv) { return; }
		var i = kv.indexOf('='), k = i < 0 ? kv : kv.slice(0, i), v = i < 0 ? '' : kv.slice(i + 1);
		try { k = decodeURIComponent(k.replace(/\+/g, ' ')); v = decodeURIComponent(v.replace(/\+/g, ' ')); } catch (e) {}
		q[k] = v;
	});
	var t, known = true;
	if ('page_id' in q || 'p' in q) { t = M.c[q.page_id || q.p]; }
	else if ('attachment_id' in q) { t = M.a[q.attachment_id]; }
	else if ('cat' in q) { t = M.k[q.cat]; }
	else if ('m' in q && /^\d{6}$/.test(q.m)) { t = q.m.slice(0, 4) + '/' + q.m.slice(4) + '/'; }
	else if ('s' in q) { t = 'search/?q=' + encodeURIComponent(q.s); }
	else if ('author' in q || 'feed' in q || 'paged' in q) { t = ''; }
	else { known = false; }
	if (!known) { return; }
	if (t === undefined) { t = '404.html'; }
	if (t === '') {
		try { history.replaceState(null, '', location.pathname + location.hash); } catch (e) {}
		return;
	}
	location.replace(t + location.hash);
})();
</script>
"""


def shell_from(ctx, name="welcome"):
    return BeautifulSoup((ctx.out / name / "index.html").read_text(encoding="utf8"), "lxml")


def clean_shell(soup, title, inner_html, body_class_drop=("single", "single-post", "postid", "page", "page-template-default")):
    soup.title.string = f"{title} \u2013 TorBox"
    for el in soup.select('link[rel="canonical"]'):
        el.decompose()
    main = soup.select_one("main#main")
    main.clear()
    main.attrs = {"id": "main", "class": ["site-main"], "data-pagefind-ignore": ""}
    main.append(frag(inner_html))
    body = soup.body
    body["class"] = [c for c in body.get("class", []) if c not in body_class_drop and not c.startswith(("postid-", "page-id-", "category-"))]
    return soup


def make_extras(ctx):
    out = ctx.out
    # redirect map on the home page
    rmap = build_redirect_map(ctx)
    idx = out / "index.html"
    t = idx.read_text(encoding="utf8")
    js = REDIRECT_JS % json.dumps(rmap, separators=(",", ":"), sort_keys=True)
    t = t.replace("<head>", "<head>\n" + js, 1)
    idx.write_text(t, encoding="utf8")
    json.dump(rmap, open(out.parent / "tools" / "data" / "redirect-map.json", "w"), indent=1, sort_keys=True)

    # /search/
    s = shell_from(ctx)
    for sec in s.select("section.widget_search"):
        sec.decompose()
    clean_shell(s, "Search", '<article class="page type-page hentry"><header class="entry-header"><h1 class="entry-title">Search</h1></header>'
                '<div class="entry-content"><p>Search across all pages, posts and archived comments of this site. '
                'The search runs entirely in your browser; no query is sent to any server.</p>'
                '<div id="search-page" class="search-page-host"></div>'
                '<noscript><p>Search requires JavaScript.</p></noscript></div></article>')
    (out / "search").mkdir(exist_ok=True)
    (out / "search" / "index.html").write_text(str(s), encoding="utf8")

    # 404.html (absolute paths from the site root, because it is served at any URL)
    s = shell_from(ctx)
    clean_shell(s, "Page not found",
                '<article class="page type-page hentry"><header class="entry-header"><h1 class="entry-title">Page not found</h1></header>'
                '<div class="entry-content"><p>The page you requested does not exist on this site. It may have been moved or removed.</p>'
                '<ul><li><a href="../">Go to the home page</a></li><li><a href="../search/">Search the site</a></li>'
                f'<li><a href="{ORIG_REPO}" rel="noopener noreferrer">TorBox on GitHub</a></li></ul></div></article>')
    s.head.append(frag('<meta name="robots" content="noindex">'))
    s.body.append(frag("<script>/* Old downloads (/data/...) now live on the download server. */"
                       "(function(){var m=location.pathname.match(/\\/data\\/([^\\/]+)$/);"
                       "if(m){location.replace('" + DL_TOKEN + "'+m[1]);}})();</script>"))
    h = str(s)
    h = re.sub(r"(?<=[\"'(])\.\./", PATH_TOKEN, h)
    h = re.sub(r"(?<=, )\.\./", PATH_TOKEN, h)
    h = h.replace('href="./"', f'href="{PATH_TOKEN}"')
    (out / "404.html").write_text(h, encoding="utf8")


def strip_gps_exif(out):
    """Remove the EXIF segment from JPEGs that carry GPS coordinates (without recompressing the image).
    A JPEG is only touched if its orientation is normal (1), so its display is not altered."""
    import struct
    done, skipped = [], []
    for f in out.rglob("*"):
        if f.suffix.lower() not in (".jpg", ".jpeg"):
            continue
        b = f.read_bytes()
        if b[:2] != b"\xff\xd8":
            continue
        i = 2
        while i + 4 < len(b) and b[i] == 0xFF and b[i + 1] != 0xDA:
            m, L = b[i + 1], struct.unpack(">H", b[i + 2:i + 4])[0]
            if m == 0xE1 and b[i + 4:i + 10] == b"Exif\0\0":
                t = b[i + 10:i + 2 + L]
                e = "<" if t[:2] == b"II" else ">"
                off = struct.unpack(e + "I", t[4:8])[0]
                n = struct.unpack(e + "H", t[off:off + 2])[0]
                tags = {}
                for k in range(n):
                    tag, _typ, _cnt, val = struct.unpack(e + "HHIH", t[off + 2 + 12 * k:off + 12 + 12 * k])
                    tags[tag] = val
                if 0x8825 in tags:
                    if tags.get(0x112, 1) == 1:
                        f.write_bytes(b[:i] + b[i + 2 + L:])
                        done.append(str(f.relative_to(out)))
                    else:
                        skipped.append(str(f.relative_to(out)))
                break
            i += 2 + L
    return done, skipped


# ---------------------------------------------------------------- main

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--mirror", required=True)
    ap.add_argument("--webroot", required=True)
    ap.add_argument("--out", required=True)
    args = ap.parse_args()
    ctx = Ctx(args)
    out = ctx.out
    # keep the site's own files (assets/, data/ of the repo) that do not come from the mirror
    keep = {}
    for sub in ("assets", "data", "robots.txt", ".nojekyll"):
        if (out / sub).exists():
            keep[sub] = out / sub
    tmp_keep = Path(str(out) + ".keep")
    if tmp_keep.exists():
        shutil.rmtree(tmp_keep)
    tmp_keep.mkdir()
    for sub, p in keep.items():
        shutil.move(str(p), str(tmp_keep / sub))
    if out.exists():
        shutil.rmtree(out)
    shutil.copytree(args.mirror, out, symlinks=False)
    for sub in os.listdir(tmp_keep):
        dest = out / sub
        src = tmp_keep / sub
        if dest.exists() and dest.is_dir():
            shutil.copytree(src, dest, dirs_exist_ok=True)
        else:
            shutil.move(str(src), str(dest))
    shutil.rmtree(tmp_keep, ignore_errors=True)

    # 1) file names: strip "?ver=..." and extensions added by wget
    for p in sorted(out.rglob("*"), key=lambda x: -len(str(x))):
        if "?" in p.name and p.is_file():
            clean = p.name.split("?")[0]
            if clean and not (p.parent / clean).exists():
                p.rename(p.parent / clean)
            else:
                p.unlink()
    # 2) discarded pages: author archives
    shutil.rmtree(out / "author", ignore_errors=True)
    for d in list(out.iterdir()):
        if d.is_dir() and d.name.startswith("⚠"):
            shutil.rmtree(d)                          # "TorBox Project Discontinued" notice
    # 3) slugs with non-ASCII characters
    for old, new in SLUG_OVERRIDE.items():
        if (out / old).exists():
            (out / old).rename(out / new)

    # 4) the original PGP key already comes with the mirror (data/); any other key from earlier runs is removed
    for stale in (out / "data").glob("zotil-*.asc"):
        stale.unlink()

    # 5) self-hosted remote images
    for url, dest in EXTERNAL_IMAGES.items():
        src = HERE / "data" / "external" / Path(dest).name
        (out / dest).parent.mkdir(parents=True, exist_ok=True)
        shutil.copy(src, out / dest)

    # 6) HTML
    pages = sorted(out.rglob("*.html"))
    for pth in pages:
        process_page(ctx, pth)

    # 7) CSS/JS with local absolute URLs
    for pth in list(out.rglob("*.css")) + list(out.rglob("*.js")):
        if "pagefind" in pth.parts:
            continue
        try:
            t = pth.read_text(encoding="utf8")
        except UnicodeDecodeError:
            continue
        if HOST_RE.search(t):
            d = str(pth.parent.relative_to(out))
            pth.write_text(rewrite_css_text(t, "" if d == "." else d), encoding="utf8")

    # 8) referenced files that wget did not download: copy from the webroot
    missing = []
    for r in sorted(ctx.refs):
        tgt = out / r
        if r == "" or (tgt.exists() and (tgt.is_file() or (tgt / "index.html").exists())):
            continue
        if r.endswith("/"):
            missing.append(r)
            continue
        src = ctx.webroot / r
        if src.is_file():
            tgt.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy(src, tgt)
        else:
            missing.append(r)
    make_extras(ctx)
    gps_done, gps_skipped = strip_gps_exif(out)
    print("GPS EXIF removed from:", gps_done, "| left untouched (different orientation):", gps_skipped)
    json.dump({"missing": missing, "retired_downloads": ctx.retired, "dead_links": ctx.dead, "unwrapped": ctx.unwrapped},
              open(out.parent / ".postprocess-report.json", "w"), indent=1)
    print(f"pages={len(pages)} refs={len(ctx.refs)} missing={len(missing)} dead={len(ctx.dead)} unwrapped={ctx.unwrapped}")
    for m in missing[:40]:
        print("  MISSING", m)


if __name__ == "__main__":
    sys.exit(main())
