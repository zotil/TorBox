[![GitHub top language](https://img.shields.io/github/languages/top/radio24/torbox.svg?style=flat-square)](https://github.com/radio24/torbox/search?l=Shell)
[![License](https://img.shields.io/github/license/radio24/torbox.svg?style=flat-square)](https://github.com/radio24/TorBox/blob/master/LICENSE)
[![Latest Release](https://img.shields.io/github/release/radio24/torbox.svg?style=flat-square)](https://github.com/radio24/TorBox/releases/latest)
- - -
> [!NOTE]
> TorBox is actively maintained by [zotil](https://github.com/zotil).
- - -
# TorBox
TorBox is an easy-to-use, anonymising router based on Raspberry Pi, which also runs on Debian, DietPi and Ubuntu-based systems. TorBox can create a separate WiFi that routes the encrypted network data over the Tor network. However, TorBox also supports clients who are connected to a cable. As [TorBox mini](https://zotil.github.io/TorBox/torbox-mini/), it can be plugged into an USB socket and be used as a network device which sends all the traffic to the Tor network. [TorBox on a Cloud](https://zotil.github.io/TorBox/installation-torbox-on-a-cloud/) can be used on a cloud combined with an OpenVPN server -- it is literally a private VPN with tor functionality. Locally installed or on a cloud, TorBox also helps to publish data easily and safely through Onion Services. The type of client (desktop, laptop, tablet, mobile, etc.) and the operating system on the client don’t matter. For more information, visit the [TorBox website](https://zotil.github.io/TorBox/).

**TorBox Image (about 2 GB)**: [v.0.5.5 (22.03.2026)](https://github.com/zotil/TorBox/releases/download/v.0.5.5-22.03.2026/torbox-20260322-v055.img.xz) – [SHA-256 values](https://zotil.github.io/TorBox/sha-256-hashes-of-all-files/)
**TorBox mini Image (about 2 GB)**: [v.0.5.5 (22.03.2026)](https://github.com/zotil/TorBox/releases/download/v.0.5.5-22.03.2026/torbox-mini-20260322-v055.img.xz) – [SHA-256 values](https://zotil.github.io/TorBox/sha-256-hashes-of-all-files/)
**TorBox Menu only**: [v.0.5.5 (22.03.2026)](https://zotil.github.io/TorBox/data/torbox055-20260322.zip) – [SHA-256 values](https://zotil.github.io/TorBox/sha-256-hashes-of-all-files/)

![What’s it all about?](https://zotil.github.io/TorBox/wp-content/uploads/2019/01/TorBox400-e1548096878388.jpg)

### Disclaimer
**Use it at your own risk!**

TorBox is ideal for providing additional protection for the entire data stream and overcoming censorship. However, **anonymity is hard to get – solely using Tor doesn’t guarantee it**. [Traffic correlation analysis](https://zotil.github.io/TorBox/deanonymisation-through-traffic-correlation-analysis/), malware, cookies, Java, Flash, Javascript and more will most certainly compromise your anonymity. Even the people from the [Tor Project themselves state](https://2019.www.torproject.org/about/overview.html.en#stayinganonymous) that “Tor can’t solve all anonymity problems. It focuses only on protecting the transport of data.” Therefore, **it is strongly advised not to use TorBox alone, should your well-being depend on your anonymity**. In such a situation, it may be better to use [Tails](https://tails.net/). Please, [read in the FAQ more about tracking and fingerprinting in web browsers](https://zotil.github.io/TorBox/faq/).

### Quick Installation Guide
1. [Download the latest TorBox image file](https://github.com/zotil/TorBox/releases/download/v.0.5.5-22.03.2026/torbox-20260322-v055.img.xz) and [verify the integrity of the downloaded file](https://zotil.github.io/TorBox/sha-256-hashes-of-all-files/).
2. Transfer the downloaded image file on an [SD Card](https://en.wikipedia.org/wiki/Secure_Digital). Because of the significantly better compression rate, we compress the images with [xz](https://en.wikipedia.org/wiki/XZ_Util). By default, [Balena Etcher](https://etcher.balena.io/) and the [Raspberry Pi Imager](https://www.raspberrypi.com/software/) support this kind of compressed image, but sometimes one or the other fails with an error message to flash it correctly on a SD Card. In this case, [you need to decompress the .xz file first](https://kagi.com/assistant/dd4e22e4-7706-489c-b33d-4c17d02cc96d). TorBox needs at least a 16 GB SD Card.
3. Put the SD Card into your Raspberry Pi, link it with an Internet router using an Ethernet cable, or place an USB WiFi adapter in one of the Raspi's USB ports to use an existing WiFi. Afterwards, start the Raspberry Pi. During the start, the system on the SD card automatically expands over the entire free partition – user interaction, screen, and peripherals are not required yet.
4. After 2-3 minutes, when the green LED stops to flicker, connect your client to the new WiFi “**TorBox055**” (password: **CHANGE-IT**).
5. Login to the TorBox by using a [SSH client](https://zotil.github.io/TorBox/faq/) (**192.168.42.1** on a WiFi client, **192.168.43.1** on a cable client or **192.168.44.1** when connected via VPN or if you use [TorBox mini](https://zotil.github.io/TorBox/torbox-mini/) on a [Raspberry Pi Zero 2 W](https://www.raspberrypi.com/products/raspberry-pi-zero-2-w/) as a RNDIS/Ethernet Gadget) or a web browser (http://192.168.42.1 on a WiFi client, http://192.168.43.1 on a cable client or http://192.168.44.1 when connected via VPN or when using TorBox mini; username: **torbox** / password: **CHANGE-IT**).
6. After [seeing a welcome screen and answering some initial questions during the first start-up](https://zotil.github.io/TorBox/first-start-up-dialogue/), you should see the [TorBox Main Menu](https://zotil.github.io/TorBox/main-menu/). Immediately, you should **change the default passwords** (the associated entries are placed in the [configuration sub-menu](https://zotil.github.io/TorBox/submenu-configuration-update/).

At least a **[Raspberry Pi 3 Model B+](https://www.raspberrypi.org/products/raspberry-pi-3-model-b-plus/)**, a **[Raspberry Pi 4 Model B](https://www.raspberrypi.org/products/raspberry-pi-4-model-b/)** or a **[Raspberry Pi 5](https://www.raspberrypi.com/products/raspberry-pi-5/)** is recommended. We also offer [installation script for other systems](https://zotil.github.io/TorBox/installation/), which might run on older (32bit) or other hardware platforms.

Do you have additional questions? Check out our [Documentation](https://zotil.github.io/TorBox/main-menu/), our [FAQ on the TorBox website](https://zotil.github.io/TorBox/faq/) or [contact us](mailto:zotil.dev@proton.me).

[![Start-up instructions](https://zotil.github.io/TorBox/wp-content/uploads/2025/12/TorBox-A5-RPI5-055-e1766508531857.png)](https://zotil.github.io/TorBox/wp-content/uploads/2025/12/TorBox-A5-RPI5-055.png)

### Features
* TorBox routes all your network data through the Tor network. At the same time, TorBox acts as an external firewall and prevents IP leakage. It also can block insecure HTTP requests. However, optionally, [TorBox can also exclude domains](https://zotil.github.io/TorBox/sub-menu-danger-zone/) from be routed through the Tor network.
* With a menu system that can be accessed by a [SSH client](https://zotil.github.io/TorBox/faq/) or a web browser, TorBox provides a user-friendly interface.
* TorBox supports Internet access via cable (Ethernet), WiFi, tethering devices, [cellular links](https://zotil.github.io/TorBox/torbox-cellular/), USB dongles (`wlan1`/`eth1`/`ppp0`/`usb0`), and VPN connections (`tun0`).
* The clients can connect TorBox via WiFi (in most cases, **an additional USB WiFi adapter is necessary**) and cable (simultaneously; see [here](https://zotil.github.io/TorBox/main-menu/)), plugged into an USB socket as a network device ([TorBox mini](https://zotil.github.io/TorBox/torbox-mini/)) or as a private VPN with tor functionality ([TorBox on a Cloud](https://zotil.github.io/TorBox/installation-torbox-on-a-cloud/)).
* It easily overcomes [captive portals](https://en.wikipedia.org/wiki/Captive_portal) and offers, if necessary, measures against “disconnect when idle features” (sometimes seen with WiFis in airports, hotels, coffee houses, etc.).
* TorBox supports [OBFS4](https://2019.www.torproject.org/docs/pluggable-transports.html), [Meek-Azure and Snowflake](https://tb-manual.torproject.org/circumvention/) bridges, which help overcome censorship ([with an easy to use interface](https://zotil.github.io/TorBox/submenu-countermeasures-troubleshooting/)).
* Also, TorBox supports [Onion Services](https://community.torproject.org/onion-services/) which allow easily and securely sharing of data through Tor and chatting securely accessing an .onion domain, even if TorBox is located behind firewalls, network address translators or placed in a censoring country while preserving the security and anonymity of both parties.
* If you have a public IP address, 24/7 Internet connectivity over a long time, and a bandwidth of at least 1 Mbps, TorBox can provide a bridge relay, easily configurable via a user-friendly interface [to allow censored users access to the open Internet](https://blog.torproject.org/run-tor-bridges-defend-open-internet).
* It provides [SOCKS v5 proxy functionality](https://en.wikipedia.org/wiki/SOCKS) on ports 9050 (standard) and 9052 (with [destination address stream isolation](https://tails.boum.org/contribute/design/stream_isolation/)).
* It allows easy access to .onion websites without client configuration (Chrome and Chromium), with [a minor change in the settings](https://zotil.github.io/TorBox/faq/) (Brave) or [via SOCKS v5 proxy](https://zotil.github.io/TorBox/faq/#SOCKS) (Mullvad and Firefox).

### Alternative installation method with the TorBox installation script
Alternatively, you can download the latest version of [Raspberry Pi OS Lite (64-bit)](https://www.raspberrypi.com/software/operating-systems/#raspberry-pi-os-64-bit) or [Raspberry Pi OS Lite (32-bit)](https://www.raspberrypi.com/software/operating-systems/), or use the [Raspberry Pi Imager](https://github.com/raspberrypi/rpi-imager/releases) and choose, under **Operating System**, **Raspberry Pi OS (other)**, the **Raspberry Pi OS Lite (64-bit)** or **Raspberry Pi OS Lite (32-bit)** image. Starting up the Raspberry Pi OS for the first time, the first-boot wizard enforces the creation of a new user account → use **torbox** and your preferred password. Optionally, localize your installation with `raspi-config`. Ensure stable Internet connectivity, download, and execute our installation script (for options, use `--help`):
```bash
cd
wget https://raw.githubusercontent.com/radio24/TorBox/master/install/run_install.sh
chmod a+x run_install.sh
./run_install.sh
```
\
See [here](https://zotil.github.io/TorBox/installation/) for more detailed information and [installation scripts for other systems](https://zotil.github.io/TorBox/installation/#others), which might run on different hardware platforms.

### TorBox mini on a Raspberry Pi Zero 2 W ###
Sometimes, a [Raspberry Pi 3 Model B+](https://www.raspberrypi.org/products/raspberry-pi-3-model-b-plus/), a [Raspberry Pi 4 Model B](https://www.raspberrypi.org/products/raspberry-pi-4-model-b/) or a [Raspberry Pi 5](https://www.raspberrypi.com/products/raspberry-pi-5/) is too big or expensive to use, and a [TorBox on a cloud](https://zotil.github.io/TorBox/installation-torbox-on-a-cloud/) is not a feasible solution. Wouldn’t it be nice if TorBox would be smaller and stealthier? That’s precisely the aim of the TorBox mini. You can find the detailed installation instructions [here](https://zotil.github.io/TorBox/torbox-mini/).

![TorBox mini in action! It is connected via a USB-A female - USB-C male adapter to a MacBook Pro.](https://zotil.github.io/TorBox/wp-content/uploads/2024/04/IMG_8288-e1712487498369.jpg)

### Building from scratch
All you need to run TorBox on your Raspberry Pi is the image file. However, if you want to build it from scratch, whether you like to implement it to an existing system, to another hardware, respectively another operating system, or you don’t trust an image file, which you didn’t bundle of your own, then check out our detailed manual [for a Raspberry Pi with Raspberry Pi OS Lite](https://zotil.github.io/TorBox/i-want-to-build-it-from-scratch-pre-v-0-23/).

### I want to help...
GREAT! There is a lot to improve and fix (security of the entire system, graphical menu, cool logos ...). We are searching for people who want to help, and **we need your feedback** to improve the system. Also, look out for our [Job Postings](https://github.com/radio24/TorBox/discussions/categories/job-posting).You can also [donate to the Tor Project](https://donate.torproject.org) -- without it, TorBox would not exist.

### Contact
* [TorBox website](https://zotil.github.io/TorBox/)
* Email: [zotil.dev@proton.me](mailto:zotil.dev@proton.me)
* Security issues and the maintainer's PGP key: see [SECURITY.md](SECURITY.md)
