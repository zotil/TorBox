#!/usr/bin/env python3
"""Local server that imitates GitHub Pages under a subpath (default /TorBox/), including the 404 page.

    python tools/serve.py --dist dist --prefix /TorBox/ --port 8099
"""
import argparse
import http.server
import mimetypes
import os
import urllib.parse
from pathlib import Path


def make_handler(dist, prefix):
    class H(http.server.BaseHTTPRequestHandler):
        server_version = "pages-sim"

        def log_message(self, fmt, *a):
            pass

        def send_file(self, path, status=200):
            data = path.read_bytes()
            ctype = mimetypes.guess_type(str(path))[0] or "application/octet-stream"
            if path.suffix in (".html", ".js", ".css", ".xml", ".txt", ".json"):
                ctype += "; charset=utf-8"
            self.send_response(status)
            self.send_header("Content-Type", ctype)
            self.send_header("Content-Length", str(len(data)))
            self.end_headers()
            if self.command != "HEAD":
                self.wfile.write(data)

        def do_HEAD(self):
            self.do_GET()

        def do_GET(self):
            p = urllib.parse.urlsplit(self.path)
            path = urllib.parse.unquote(p.path)
            if path == prefix.rstrip("/"):
                self.send_response(301)
                self.send_header("Location", prefix)
                self.end_headers()
                return
            if path.startswith(prefix):
                rel = path[len(prefix):]
                f = (dist / rel).resolve()
                if dist.resolve() in f.parents or f == dist.resolve():
                    if f.is_dir():
                        if not path.endswith("/"):
                            self.send_response(301)
                            self.send_header("Location", path + "/" + ("?" + p.query if p.query else ""))
                            self.end_headers()
                            return
                        f = f / "index.html"
                    if f.is_file():
                        return self.send_file(f)
            nf = dist / "404.html"
            if nf.exists():
                return self.send_file(nf, 404)
            self.send_response(404)
            self.end_headers()
    return H


if __name__ == "__main__":
    ap = argparse.ArgumentParser()
    ap.add_argument("--dist", default="dist")
    ap.add_argument("--prefix", default="/TorBox/")
    ap.add_argument("--port", type=int, default=8099)
    a = ap.parse_args()
    srv = http.server.ThreadingHTTPServer(("127.0.0.1", a.port), make_handler(Path(a.dist), a.prefix))
    print(f"http://127.0.0.1:{a.port}{a.prefix}")
    srv.serve_forever()
