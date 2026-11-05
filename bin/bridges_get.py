#!/usr/bin/python3

# This file is part of TorBox, an easy to use anonymizing router based on Raspberry Pi.
# Copyright (C) 2026 radio_24
# Contact: anonym@torbox.ch
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it is useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program. If not, see <http://www.gnu.org/licenses/>.
#
# DESCRIPTION
# This file fetches new bridges from the Tor Project's bridge distribution
# (rdsys, moat "circumvention" API). The return values are one bridge line per
# line, for example:
# obfs4 <IP address>:<Port> <Fingerprint> <Certificate> <iat-mode>
# or an error code (see below).
#
# IMPORTANT
# Since BridgeDB was replaced by rdsys (September 2024), bridges are no longer
# requested with a CAPTCHA. Without a country, the "defaults" endpoint is used;
# with a country, the "settings" endpoint returns the bridges recommended for
# that country. Built-in bridges are only used for Snowflake.
#
# SYNTAX
# Usage: bridges_get.py [OPTIONS]
#
# Options:
#   -n, --network TEXT  Force to get bridges over specific network
#   -c, --country TEXT  Circumvention country setting - <TEXT> has to be in lowercase
#   --snowflake         Get snowflake bridges instead of obfs4
#   --help              Show this message and exit.
#
# ERROR CODES:
# -1: Network error
# -2: Bridges not found


import sys

import click
import requests

MOAT_URL = "https://bridges.torproject.org/moat/circumvention"
HEADERS = {"Content-Type": "application/vnd.api+json"}
TIMEOUT = 60


def get_proxy(network=''):
    # Tor Socks Proxy
    proxy = {
        "http": "socks5h://127.0.0.1:9050",
        "https": "socks5h://127.0.0.1:9050",
    }

    if not network:
        # Check if Tor proxy is up, otherwise set inet network
        try:
            requests.get("https://torproject.org", proxies=proxy, timeout=5)
        except requests.exceptions.RequestException:
            network = 'inet'

    if network == "inet":
        proxy = {
            "http": "",
            "https": "",
        }
    return proxy


def moat_request(endpoint, data, proxy):
    """POST to a moat circumvention endpoint and return the list of settings."""
    try:
        r = requests.post(f"{MOAT_URL}/{endpoint}", json=data, proxies=proxy,
                          headers=HEADERS, timeout=TIMEOUT)
        r.raise_for_status()
        return r.json().get("settings") or []
    except (requests.exceptions.RequestException, ValueError):
        print(-1)
        sys.exit(1)


def pick_bridges(settings, transport, sources):
    """Return the unique bridge lines of a transport from the given sources."""
    bridges = []
    for setting in settings:
        b = setting.get("bridges") or {}
        if b.get("type") == transport and b.get("source") in sources:
            for line in b.get("bridge_strings") or []:
                if line not in bridges:
                    bridges.append(line)
    return bridges


def get_bridges(network, country, snowflake):
    proxy = get_proxy(network)
    transport = "snowflake" if snowflake else "obfs4"
    # obfs4: only newly distributed bridges; snowflake: built-in lines are fine
    sources = ("bridgedb", "builtin") if snowflake else ("bridgedb",)

    bridges = []
    if country:
        settings = moat_request("settings", {"country": country, "transports": [transport]}, proxy)
        bridges = pick_bridges(settings, transport, sources)

    # No country, or no special settings for that country: use the defaults
    if not bridges:
        settings = moat_request("defaults", {"transports": [transport]}, proxy)
        bridges = pick_bridges(settings, transport, sources)

    return bridges


# fmt: off
@click.command()
@click.option('--network', '-n', default='', type=str, help="Force to get bridges over specific network. Example: -n <tor|inet>")
@click.option('--country', '-c', default='', type=str, help="Circumvention country setting in lowercase")
@click.option('--snowflake', is_flag=True, default=False, show_default=True, help="Get snowflake bridges instead of obfs4")
# fmt: on
def main(network, country, snowflake):
    # Validate options
    if country and len(country) > 2:
        print("[X] Invalid country")
        sys.exit(1)

    bridges = get_bridges(network=network, country=country.lower(), snowflake=snowflake)
    if not bridges:
        print(-2)
        sys.exit(1)

    print("\n".join(bridges))


if __name__ == '__main__':
    main()
