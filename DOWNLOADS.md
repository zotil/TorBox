# Downloadable files

The download links on the site point to `DOWNLOAD_BASE_URL` (defined in `config.json`). Files are served directly
under that URL, without subfolders: `<DOWNLOAD_BASE_URL><file-name>`.

Current value: `https://github.com/zotil/TorBox/releases/download/v.0.5.5-22.03.2026/`

The images are published as **GitHub Releases assets** (release `v.0.5.5-22.03.2026`), not on a separate download server.

## Release assets

Only two files are linked, both with the SHA-256 published on the site's hashes page:

| File | Size | SHA-256 |
|---|---|---|
| `torbox-20260322-v055.img.xz` | 1.97 GB | `4A28964608D54D2541F2078981D9903E5B83DDE7FA7A8BE5DE8E0E33033A1C08` |
| `torbox-mini-20260322-v055.img.xz` | 2.00 GB | `1BE48258728AC8FB157FD2C8A2383498491C59A86EB10EB2A08C72F93538A359` |

## No longer linked

Links to all other images were removed (the text is kept): 8 `.img.xz` files (v0.5.4 and the December 2025 builds
of v0.5.5) and 13 `.gz` files (0.3.0 to 0.5.3). The 17 old `.zip` files of the menu link to
<https://github.com/radio24/TorBox/tags>.

## Already included in the repository

`site/data/`: `torbox055-20260322.zip` (6.4 MB, SHA-256 `D84E64C7…F7BE01`) and the project's original PGP key (`.asc`).

## Old `/data/...` paths

If the old domain is pointed at this site, `404.html` redirects any `/data/<file>` URL to
`DOWNLOAD_BASE_URL<file>`.
