# Security Policy

**Use it at your own risk!**

TorBox is ideal for providing additional protection for the entire data stream up to the Tor network and overcoming censorship. However, **anonymity is hard to get – solely using Tor doesn’t guarantee it**. Malware, Cookies, Java, Flash, Javascript, and more will most certainly compromise your anonymity. Even the people from [the Tor Project themselves state that](https://2019.www.torproject.org/about/overview.html.en#stayinganonymous) “Tor can’t solve all anonymity problems. It focuses only on protecting the transport of data.” Therefore, **it is strongly advised not to use TorBox if your well-being depends on your anonymity**. In such a situation, it is advisable to use [Tails](https://tails.boum.org/).

## Supported Versions / SHA-256 hashes

If you [download](https://zotil.github.io/TorBox/sha-256-hashes-of-all-files/) the TorBox image file and/or the TorBox menu package from the [TorBox website](https://zotil.github.io/TorBox/), you can check the integrity of the downloaded files with the provided SHA-256 hashes (for details, see [here](https://zotil.github.io/TorBox/sha-256-hashes-of-all-files/)).

Updates (for example, by using entry 5 in the [Update and Reset menu](https://zotil.github.io/TorBox/sub-menu-update-and-reset/)) are made only for the latest version. If possible, we describe the update path from the previous to the last version in the [Blog post to a new version](https://zotil.github.io/TorBox/category/blog/). Nevertheless, we recommend, if possible, using the new image.

## Reporting a Vulnerability

Please report security issues privately through [GitHub's private vulnerability reporting](https://github.com/zotil/TorBox/security/advisories/new), or by email to [zotil.dev@proton.me](mailto:zotil.dev@proton.me).

For encrypted email, use the maintainer's PGP key:

- Fingerprint: `F914 0269 F3EB A109 5B93  17AA 3B39 0D54 D028 1EBE`
- Key: [PUBLICKEY-zotil.asc](PUBLICKEY-zotil.asc) (also available on [keys.openpgp.org](https://keys.openpgp.org/vks/v1/by-fingerprint/F9140269F3EBA1095B9317AA3B390D54D0281EBE))

## Verifying the Maintainer's Key

The maintainer's key is certified by the former TorBox project key ([PUBLICKEY.asc](PUBLICKEY.asc), fingerprint `69E1 14C5 C446 133A 0489  A6C0 E849 2953 8341 E840`). That key is kept in the repository only to verify the maintainer's key; its email address is no longer a contact for the project. To verify it:

```
gpg --import PUBLICKEY.asc PUBLICKEY-zotil.asc
gpg --check-sigs F9140269F3EBA1095B9317AA3B390D54D0281EBE
```

Each user ID of the maintainer's key must show a valid signature (`sig!`) from `E84929538341E840`.
