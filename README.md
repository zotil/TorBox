[![GitHub top language](https://img.shields.io/github/languages/top/radio24/torbox.svg?style=flat-square)](https://github.com/radio24/torbox/search?l=Shell)
[![License](https://img.shields.io/github/license/radio24/torbox.svg?style=flat-square)](https://github.com/radio24/TorBox/blob/master/LICENSE)
[![Latest Release](https://img.shields.io/github/release/radio24/torbox.svg?style=flat-square)](https://github.com/radio24/TorBox/releases/latest)
- - -
> [!IMPORTANT]
> ## ⚠️ Project Discontinued
>
> **TorBox is no longer actively maintained.**
>
> Due to a lack of community interest and user engagement, development of this project has been discontinued. No further updates, bug fixes, or feature additions are planned.
>
> **The repository will remain publicly available as-is** — feel free to browse the code, fork it, or use it as a reference. However, please be aware that dependencies may become outdated and no support will be provided.
>
> Thank you to everyone who contributed or used TorBox over the years.
- - -
# TorBox
TorBox is an easy-to-use, anonymising router based on Raspberry Pi, which also runs on Debian, DietPi and Ubuntu-based systems. TorBox can create a separate WiFi that routes the encrypted network data over the Tor network. However, TorBox also supports clients who are connected to a cable. As [TorBox mini](https://www.torbox.ch/?page_id=3544), it can be plugged into an USB socket and be used as a network device which sends all the traffic to the Tor network. [TorBox on a Cloud](https://www.torbox.ch/?page_id=3687) can be used on a cloud combined with an OpenVPN server -- it is literally a private VPN with tor functionality. Locally installed or on a cloud, TorBox also helps to publish data easily and safely through Onion Services. The type of client (desktop, laptop, tablet, mobile, etc.) and the operating system on the client don’t matter. For more information, visit the [TorBox website](https://www.torbox.ch).

**TorBox Image (about 2 GB)**: [v.0.5.5 (22.03.2026)](https://www.torbox.ch/data/torbox-20260322-v055.img.xz) – [SHA-256 values](https://www.torbox.ch/?page_id=1128)
**TorBox mini Image (about 2 GB)**: [v.0.5.5 (22.03.2026)](https://www.torbox.ch/data/torbox-mini-20260322-v055.img.xz) – [SHA-256 values](https://www.torbox.ch/?page_id=1128)
**TorBox Menu only**: [v.0.5.5 (22.03.2026)](https://www.torbox.ch/data/torbox055-20260322.zip) – [SHA-256 values](https://www.torbox.ch/?page_id=1128)

![What’s it all about?](https://www.torbox.ch/wp-content/uploads/2019/01/TorBox400-e1548096878388.jpg)

### Disclaimer
**Use it at your own risk!**

TorBox is ideal for providing additional protection for the entire data stream and overcoming censorship. However, **anonymity is hard to get – solely using Tor doesn’t guarantee it**. [Traffic correlation analysis](https://www.torbox.ch/?p=3634), malware, cookies, Java, Flash, Javascript and more will most certainly compromise your anonymity. Even the people from the [Tor Project themselves state](https://2019.www.torproject.org/about/overview.html.en#stayinganonymous) that “Tor can’t solve all anonymity problems. It focuses only on protecting the transport of data.” Therefore, **it is strongly advised not to use TorBox alone, should your well-being depend on your anonymity**. In such a situation, it may be better to use [Tails](https://tails.net/). Please, [read in the FAQ more about tracking and fingerprinting in web browsers](https://www.torbox.ch/?page_id=112#can-tor-protect-me-against-tracking-andor-fingerprinting-in-web-browser-to-guaranty-my-anonymity-accessing-a-website).

### Quick Installation Guide
1. [Download the latest TorBox image file](https://www.torbox.ch/data/torbox-20260322-v055.img.xz) and [verify the integrity of the downloaded file](https://www.torbox.ch/?page_id=1128).
2. Transfer the downloaded image file on an [SD Card](https://en.wikipedia.org/wiki/Secure_Digital). Because of the significantly better compression rate, we compress the images with [xz](https://en.wikipedia.org/wiki/XZ_Util). By default, [Balena Etcher](https://etcher.balena.io/) and the [Raspberry Pi Imager](https://www.raspberrypi.com/software/) support this kind of compressed image, but sometimes one or the other fails with an error message to flash it correctly on a SD Card. In this case, [you need to decompress the .xz file first](https://kagi.com/assistant/dd4e22e4-7706-489c-b33d-4c17d02cc96d). TorBox needs at least a 16 GB SD Card.
3. Put the SD Card into your Raspberry Pi, link it with an Internet router using an Ethernet cable, or place an USB WiFi adapter in one of the Raspi's USB ports to use an existing WiFi. Afterwards, start the Raspberry Pi. During the start, the system on the SD card automatically expands over the entire free partition – user interaction, screen, and peripherals are not required yet.
4. After 2-3 minutes, when the green LED stops to flicker, connect your client to the new WiFi “**TorBox055**” (password: **CHANGE-IT**).
5. Login to the TorBox by using a [SSH client](https://www.torbox.ch/?page_id=112#how-can-i-access-the-torbox-menu) (**192.168.42.1** on a WiFi client, **192.168.43.1** on a cable client or **192.168.44.1** when connected via VPN or if you use [TorBox mini](https://www.torbox.ch/?page_id=3544) on a [Raspberry Pi Zero 2 W](https://www.raspberrypi.com/products/raspberry-pi-zero-2-w/) as a RNDIS/Ethernet Gadget) or a web browser (http://192.168.42.1 on a WiFi client, http://192.168.43.1 on a cable client or http://192.168.44.1 when connected via VPN or when using TorBox mini; username: **torbox** / password: **CHANGE-IT**).
6. After [seeing a welcome screen and answering some initial questions during the first start-up](https://www.torbox.ch/?page_id=2637), you should see the [TorBox Main Menu](https://www.torbox.ch/?page_id=775). Immediately, you should **change the default passwords** (the associated entries are placed in the [configuration sub-menu](https://www.torbox.ch/?page_id=875).

At least a **[Raspberry Pi 3 Model B+](https://www.raspberrypi.org/products/raspberry-pi-3-model-b-plus/)**, a **[Raspberry Pi 4 Model B](https://www.raspberrypi.org/products/raspberry-pi-4-model-b/)** or a **[Raspberry Pi 5](https://www.raspberrypi.com/products/raspberry-pi-5/)** is recommended. We also offer [installation script for other systems](https://www.torbox.ch/?page_id=1168), which might run on older (32bit) or other hardware platforms.

Do you have additional questions? Check out our [Documentation](https://www.torbox.ch/?page_id=775), our [FAQ on the TorBox website](https://www.torbox.ch/?page_id=112) or [contact us](mailto:anonym@torbox.ch).

[![Start-up instructions](https://www.torbox.ch/wp-content/uploads/2025/12/TorBox-A5-RPI5-055-e1766508531857.png)](https://www.torbox.ch/wp-content/uploads/2025/12/TorBox-A5-RPI5-055.png)

### Features
* TorBox routes all your network data through the Tor network. At the same time, TorBox acts as an external firewall and prevents IP leakage. It also can block insecure HTTP requests. However, optionally, [TorBox can also exclude domains](https://www.torbox.ch/?page_id=3445) from be routed through the Tor network.
* With a menu system that can be accessed by a [SSH client](https://www.torbox.ch/?page_id=112#how-can-i-access-the-torbox-menu) or a web browser, TorBox provides a user-friendly interface.
* TorBox supports Internet access via cable (Ethernet), WiFi, tethering devices, [cellular links](https://www.torbox.ch/?page_id=1030), USB dongles (`wlan1`/`eth1`/`ppp0`/`usb0`), and VPN connections (`tun0`).
* The clients can connect TorBox via WiFi (in most cases, **an additional USB WiFi adapter is necessary**) and cable (simultaneously; see [here](https://www.torbox.ch/?page_id=775)), plugged into an USB socket as a network device ([TorBox mini](https://www.torbox.ch/?page_id=3544)) or as a private VPN with tor functionality ([TorBox on a Cloud](https://www.torbox.ch/?page_id=3687)).
* It easily overcomes [captive portals](https://en.wikipedia.org/wiki/Captive_portal) and offers, if necessary, measures against “disconnect when idle features” (sometimes seen with WiFis in airports, hotels, coffee houses, etc.).
* TorBox supports [OBFS4](https://2019.www.torproject.org/docs/pluggable-transports.html), [Meek-Azure and Snowflake](https://tb-manual.torproject.org/circumvention/) bridges, which help overcome censorship ([with an easy to use interface](https://www.torbox.ch/?page_id=797)).
* Also, TorBox supports [Onion Services](https://community.torproject.org/onion-services/) which allow easily and securely sharing of data through Tor and chatting securely accessing an .onion domain, even if TorBox is located behind firewalls, network address translators or placed in a censoring country while preserving the security and anonymity of both parties.
* If you have a public IP address, 24/7 Internet connectivity over a long time, and a bandwidth of at least 1 Mbps, TorBox can provide a bridge relay, easily configurable via a user-friendly interface [to allow censored users access to the open Internet](https://blog.torproject.org/run-tor-bridges-defend-open-internet).
* It provides [SOCKS v5 proxy functionality](https://en.wikipedia.org/wiki/SOCKS) on ports 9050 (standard) and 9052 (with [destination address stream isolation](https://tails.boum.org/contribute/design/stream_isolation/)).
* It allows easy access to .onion websites without client configuration (Chrome and Chromium), with [a minor change in the settings](https://www.torbox.ch/?page_id=112#when-i-start-the-tor-bowser-or-when-i-open-a-window-with-tor-on-brave-both-running-on-a-client-device-of-the-torbox-i-cannot-connect-to-the-tor-network-the-same-happens-if-im-using-tails-behind-a-torbox) (Brave) or [via SOCKS v5 proxy](https://www.torbox.ch/?page_id=112#SOCKS) (Mullvad and Firefox).

### Alternative installation method with the TorBox installation script
Alternatively, you can download the latest version of [Raspberry Pi OS Lite (64-bit)](https://www.raspberrypi.com/software/operating-systems/#raspberry-pi-os-64-bit) or [Raspberry Pi OS Lite (32-bit)](https://www.raspberrypi.com/software/operating-systems/), or use the [Raspberry Pi Imager](https://github.com/raspberrypi/rpi-imager/releases) and choose, under **Operating System**, **Raspberry Pi OS (other)**, the **Raspberry Pi OS Lite (64-bit)** or **Raspberry Pi OS Lite (32-bit)** image. Starting up the Raspberry Pi OS for the first time, the first-boot wizard enforces the creation of a new user account → use **torbox** and your preferred password. Optionally, localize your installation with `raspi-config`. Ensure stable Internet connectivity, download, and execute our installation script (for options, use `--help`):
```bash
cd
wget https://raw.githubusercontent.com/radio24/TorBox/master/install/run_install.sh
chmod a+x run_install.sh
./run_install.sh
```
\
See [here](https://www.torbox.ch/?page_id=1168) for more detailed information and [installation scripts for other systems](https://www.torbox.ch/?page_id=1168#others), which might run on different hardware platforms.

### TorBox mini on a Raspberry Pi Zero 2 W ###
Sometimes, a [Raspberry Pi 3 Model B+](https://www.raspberrypi.org/products/raspberry-pi-3-model-b-plus/), a [Raspberry Pi 4 Model B](https://www.raspberrypi.org/products/raspberry-pi-4-model-b/) or a [Raspberry Pi 5](https://www.raspberrypi.com/products/raspberry-pi-5/) is too big or expensive to use, and a [TorBox on a cloud](https://www.torbox.ch/?page_id=3687) is not a feasible solution. Wouldn’t it be nice if TorBox would be smaller and stealthier? That’s precisely the aim of the TorBox mini. You can find the detailed installation instructions [here](https://www.torbox.ch/?page_id=3544).

![TorBox mini in action! It is connected via a USB-A female - USB-C male adapter to a MacBook Pro.](https://www.torbox.ch/wp-content/uploads/2024/04/IMG_8288-e1712487498369.jpg)

### Building from scratch
All you need to run TorBox on your Raspberry Pi is the image file. However, if you want to build it from scratch, whether you like to implement it to an existing system, to another hardware, respectively another operating system, or you don’t trust an image file, which you didn’t bundle of your own, then check out our detailed manual [for a Raspberry Pi with Raspberry Pi OS Lite](https://www.torbox.ch/?page_id=205).

### I want to help...
GREAT! There is a lot to improve and fix (security of the entire system, graphical menu, cool logos ...). We are searching for people who want to help, and **we need your feedback** to improve the system. Also, look out for our [Job Postings](https://github.com/radio24/TorBox/discussions/categories/job-posting).You can also [donate to the Tor Project](https://donate.torproject.org) -- without it, TorBox would not exist.

### Contact
* [TorBox website](https://www.torbox.ch)
* [TorBox email](mailto:anonym@torbox.ch)

For secure email communication, we are using for the [TorBox email](mailto:anonym@torbox.ch) [Protonmail](https://protonmail.com). All messages between Protonmail users are automatically [end-to-end encrypted](https://protonmail.com/blog/what-is-end-to-end-encryption/). Additionally, all messages in Protonmail inboxes are protected with [PGP encryption](https://en.wikipedia.org/wiki/Pretty_Good_Privacy) to prevent Protonmail (or anyone else) from reading or sharing emails, a concept known as [zero-access encryption](https://protonmail.com/blog/zero-access-encryption/). Creating a Protonmail email address is free and takes less than a minute. With Protonmail, anyone can use PGP regardless of their technical knowledge. However, technically versed, can also use [our public PGP key](https://raw.githubusercontent.com/radio24/TorBox/master/PUBLICKEY.asc) to communicate with us:

```
-----BEGIN PGP PUBLIC KEY BLOCK-----
Version: OpenPGP.js v4.10.10
Comment: https://openpgpjs.org

xjMEXemNYRYJKwYBBAHaRw8BAQdAH22RKj/kZRqZds03njk7tSFEgrYkbeFo
PRC3CwA2JwPNI2Fub255bUB0b3Jib3guY2ggPGFub255bUB0b3Jib3guY2g+
wncEEBYKAB8FAl3pjWEGCwkHCAMCBBUICgIDFgIBAhkBAhsDAh4BAAoJEOhJ
KVODQehAkY8A/A7vPC+6nPaGBiv7P6wryQ+THA97uEwRK0Rsx3TYlKHuAQDN
M4XH5G++eqqptaEv1daJEofwOnYxahJoHzYvdfZUBM44BF3pjWESCisGAQQB
l1UBBQEBB0Cp+yT4Ec5kmGaGWneulB/KSgXLkkMSVaD++dC9mrcTfQMBCAfC
YQQYFggACQUCXemNYQIbDAAKCRDoSSlTg0HoQArZAQD94cT2csOWOsqqx7+q
Ps0P1Udn2/jXRbO+XbfzBzjM6wEAq4Z4g0w03KkHC3aU8/fATEnbN2+TInLV
gNKTldrMtAg=
=eGoI
-----END PGP PUBLIC KEY BLOCK-----
```

<img src="https://www.torbox.ch/wp-content/uploads/2021/08/pgp_asc-e1628022322939.jpeg" width="32" height="32">OpenPGP key file: [publickey.anonym@torbox.ch-69e114c5c446133a0489a6c0e84929538341e840.asc](https://torbox.ch/data/publickey.anonym@torbox.ch-69e114c5c446133a0489a6c0e84929538341e840.asc)
