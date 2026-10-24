# TorBox – static website (`website` branch)

This content lives on the **orphan `website` branch** of the `zotil/TorBox` repository (at the root of that branch,
unrelated to the history of `master`, so that it does not increase the size of the `master` zip downloaded by
`run_install.sh` and `menu-update`). It is published at <https://zotil.github.io/TorBox/>.

Static version (GitHub Pages) of the TorBox site, generated from the WordPress site at torbox.ch.
The final site **makes no requests to third parties**: fonts, scripts, styles and images are self-hosted,
there is no analytics and no embeds, and every page carries a restrictive Content-Security-Policy (`default-src 'self'`).

## Layout

| Path | Contents |
|---|---|
| `site/` | The cleaned site (HTML, CSS, JS, images). This is the published source. It contains `@@…@@` tokens. |
| `config.json` | The **only place** where `SITE_BASE_URL` and `DOWNLOAD_BASE_URL` are configured. |
| `tools/build.py` | Copies `site/` to `dist/`, substitutes the tokens, generates `sitemap.xml` and `robots.txt`. |
| `tools/check_no_external.py` | Fails if any page loads third-party resources. |
| `tools/postprocess.py`, `tools/mirror.sh` | Pipeline that turned the WordPress mirror into `site/` (only needed to regenerate from WordPress). |
| `tools/data/` | ID→slug map exported from the database, redirect map, self-hosted external image. |
| `package.json` | Pins the Pagefind version (search). |
| `.github/workflows/website.yml` | Workflow of the `website` branch: builds, checks, indexes with Pagefind and deploys to Pages. Runs on every push to `website`. |
| `DOWNLOADS.md` | Files served from `DOWNLOAD_BASE_URL`. |

## Configuration (single place)

`config.json`:

```json
{
  "SITE_BASE_URL": "https://zotil.github.io/TorBox/",
  "DOWNLOAD_BASE_URL": "https://github.com/zotil/TorBox/releases/download/v.0.5.5-22.03.2026/"
}
```

* `SITE_BASE_URL`: public URL of the site. Used in `<link rel="canonical">`, `sitemap.xml`, `robots.txt` and for the
  absolute paths of `404.html`. The rest of the site uses relative links and works under any subpath or domain.
* `DOWNLOAD_BASE_URL`: base URL of the images (`.img.xz`). See `DOWNLOADS.md`.

## Build and test locally

```sh
npm ci                                   # installs Pagefind (pinned version)
python3 tools/build.py                   # site/ -> dist/
npx --no-install pagefind --site dist    # search index
python3 tools/check_no_external.py dist  # no third-party loads
python3 tools/serve.py --dist dist --prefix /TorBox/ --port 8099   # http://127.0.0.1:8099/TorBox/
```

`tools/serve.py` imitates GitHub Pages under the `/TorBox/` subpath, including the 404 page.

## Deployment

Every push to the `website` branch runs: `build.py` → `check_no_external.py` → Pagefind → Pages. Actions are pinned by SHA.
One-time setup on GitHub:

1. *Settings → Pages → Build and deployment → Source: GitHub Actions*.
2. *Settings → Environments → github-pages → Deployment branches and tags*: by default only the default branch (`master`) is allowed.
   Choose *Selected branches and tags* and add the rule **Branch `website`** (or *No restriction*). Without this the `deploy` job
   fails with "Branch "website" is not allowed to deploy to github-pages".

The workflow has no `workflow_dispatch` trigger because GitHub only lists such triggers when the file exists on the default branch.
To redeploy without changes: *Re-run jobs* in Actions, or an empty commit (`git commit --allow-empty -m redeploy && git push`).
Workflows on `master` (e.g. `main.yml`) do not run for this branch, and this content does not enter `master`.

## Custom domain

1. In `config.json`, change `SITE_BASE_URL` to the domain (for example `https://torbox.ch/`), with a trailing slash.
2. (With a custom domain the site is served at the domain root, not under `/TorBox/`; links are relative, so nothing else needs to change.)
3. In DNS: `A` records for the apex domain pointing to `185.199.108.153`, `185.199.109.153`, `185.199.110.153`, `185.199.111.153`
   (and `AAAA` `2606:50c0:8000::153`, `…8001::153`, `…8002::153`, `…8003::153`), or a `CNAME` for `www` pointing to `zotil.github.io`.
4. In the repository: *Settings → Pages → Custom domain*, enter the domain, and enable *Enforce HTTPS* once the certificate is issued.
5. Push the `config.json` change. The redirects for `/?page_id=N`, `/?p=N`, etc. (inline in `index.html`)
   and the `/data/*` redirect (in `404.html`) keep working without further changes.
6. Check <https://docs.github.com/pages> for the current IP addresses before configuring DNS.

## Dynamic WordPress features and their replacement

| WordPress | Static |
|---|---|
| Search | Pagefind (sidebar widget and `/search/` page); the index is built in the workflow |
| Comments | Removed; the 162 approved comments are kept as text ("Archived comments") with a link to GitHub Discussions |
| `/?page_id=N`, `/?p=N`, `/?attachment_id=N`, `/?cat=N`, `/?m=YYYYMM`, `/?s=` | JavaScript map in `index.html` (generated from the database) |
| 404 | Custom `404.html` |
| Downloads `/data/*` | Redirected to `DOWNLOAD_BASE_URL` from `404.html` |
