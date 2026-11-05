#!/bin/bash
# shellcheck disable=SC2001

# This file is a part of TorBox, an easy to use anonymizing router based on Raspberry Pi.
# Copyright (C) 2026 radio_24
# Contact: anonym@torbox.ch
# Website: https://www.torbox.ch
# Github:  https://github.com/radio24/TorBox
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
# This script prepares the freshly installed TorBox for the building of an image
#
# SYNTAX
# ./prepare_image.sh
#
#
###### SET VARIABLES ######

#Colors
RED='\033[1;31m'
GREEN='\033[1;32m'
YELLOW='\033[1;93m'
NOCOLOR='\033[0m'

#Other variables
TORRC="/etc/tor/torrc"
TORBOX_PATH="/home/torbox/torbox"
RUNFILE="$TORBOX_PATH/run/torbox.run"

# TorBox Repository
[ -z "$TORBOXMENU_FORKNAME" ] && TORBOXMENU_FORKNAME="radio24"
[ -z "$TORBOXMENU_BRANCHNAME" ] && TORBOXMENU_BRANCHNAME="master"

# Read configuration from run/torbox.run
TORBOX_MINI=$(grep "^TORBOX_MINI=.*" ${RUNFILE} | sed "s/.*=//g")
ON_A_CLOUD=$(grep "^ON_A_CLOUD=.*" ${RUNFILE} | sed "s/.*=//g")

##############################
######## FUNCTIONS ###########

# include lib
.  /home/torbox/torbox/lib/torbox.lib

# Is the automatic counteractions feature activated?
if pgrep -f "log_check.py" ; then
  clear
	LOGCHECK_STRING="${GREEN} Activated!"
else
    clear
	LOGCHECK_STRING="${RED} Deactivated!"
fi

# Are bridges activated?
if grep "^UseBridges" ${TORRC}; then
	if grep -o "^Bridge obfs4 " ${TORRC}; then
			MODE_BRIDGES="${RED} OBFS4 is running - will be deactivated."
	elif grep -o "^Bridge meek_lite " ${TORRC}; then
			MODE_BRIDGES="${RED} Meek-Azure is running - will be deactivated."
	elif grep -o "^Bridge snowflake " ${TORRC}; then
				MODE_BRIDGES="${RED} Snowflake is running - will be deactivated."
	else
			MODE_BRIDGES="${GREEN} Are not running."
	fi
else
		MODE_BRIDGES="${GREEN} Are not running."
fi

# Is the Bridge Relay activated?
if grep "^BridgeRelay" ${TORRC}; then
	BRIDGE_RELAY="=${RED} Is running - will be deactivated."
else
	BRIDGE_RELAY="${GREEN} Is not running."
fi

# Are Onion Services Running?
if grep "^HiddenServiceDir" ${TORRC}; then
	MODE_OS="${RED} Are running."
else
	MODE_OS="${GREEN} Are not running."
fi

# Is the Countermeasure against a tightly configured firewall active?
if grep -o "^ReachableAddresses " ${TORRC}; then
	FIREWALL="${RED} Are running - will be deactivated."
  sudo sed -i "s/^ReachableAddresses /#ReachableAddresses /g" ${TORRC}
else
	FIREWALL="${GREEN} Are not running."
fi

# Is the Countermeasure against a disconnection when idle feature active?
	if pgrep -f "ping -q $PING_SERVER" ; then
		clear
		PING="${RED} Is running - will be deactivated."
    sudo killall ping
	else
		clear
		PING="${GREEN} Is not running."
	fi

# Snowflake version
SNOWFLAKE_VERS=$(snowflake-proxy --version 2>&1 | grep snowflake)

clear
echo -e "${YELLOW}[!] CHECK INSTALLATION${NOCOLOR}"
echo
if [ "$TORBOX_MINI" -eq "1" ]; then echo -e "${YELLOW}ATTENTION: This is a TorBox mini - installation!${NOCOLOR}"; echo; fi
if [ "$ON_A_CLOUD" -eq "1" ]; then echo -e "${YELLOW}ATTENTION: This is an installation on a cloud!${NOCOLOR}"; echo; fi
echo -e "${RED}Hostname                               :${YELLOW} $(cat /etc/hostname)${NOCOLOR}"
echo -e "${RED}Kernel version                         :${YELLOW} $(uname -a)${NOCOLOR}"
echo -e "${RED}Tor version                            :${YELLOW} $(tor -v | head -1 | sed "s/Tor version //" | cut -c1-80)${NOCOLOR}"
echo -e "${RED}Obfs4proxy version                     :${YELLOW} $(obfs4proxy --version | head -1 | sed "s/obfs4proxy-//")${NOCOLOR}"
echo -e "${RED}Snowflake                              :${YELLOW} ${SNOWFLAKE_VERS}${NOCOLOR}"
echo -e "${RED}Nyx version                            :${YELLOW} $(nyx -v | head -1 | sed "s/nyx version //")${NOCOLOR}"
echo -e "${RED}Go version                             :${YELLOW} $(/usr/local/go/bin/go version | head -1 | sed "s/go version //")${NOCOLOR}"
echo -e "${RED}Firewall countermeasures               :$FIREWALL${NOCOLOR}"
echo -e "${RED}Disconnection when idle countermeasure :$PING${NOCOLOR}"
echo -e "${RED}TorBox's automatic counteractions are  :$LOGCHECK_STRING${NOCOLOR}"
echo -e "${RED}Bridges                                :$MODE_BRIDGES${NOCOLOR}"
echo -e "${RED}Bridge Relay                           :$BRIDGE_RELAY${NOCOLOR}"
echo -e "${RED}Onion Services                         :$MODE_OS${NOCOLOR}"
echo
read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
clear
# For RaspberryPi OS based on Debian Bookworm needed
PYTHON_LIB_PATH=$(python3 -c "import sys; print(sys.path)" | cut -d ',' -f3 | sed "s/'//g" | sed "s/,//g" | sed "s/ //g")
if [ -f "$PYTHON_LIB_PATH/EXTERNALLY-MANAGED" ] ; then
  sudo rm "$PYTHON_LIB_PATH/EXTERNALLY-MANAGED"
fi
cd
# Install and check Python requirements
# How to deal with Pipfile, Pipfile.lock and requirements.txt:
# 1. Check the Pipfile --> is the package in the list?
# 2. Execute: pipenv lock (this should only be done on a test system not during installation or to prepare an image!)
# 3. Execute: pipenv requirements >requirements.txt
# 4. Execute: sudo pip install -r requirements.txt (this will update outdated packages)
# 5. Check the list of outdated packages: pip list --outdated
# Remark: we install all Python libraries globally (as root) because otherwice some programs troubling to find the library in the local environment
# NEW v.0.5.5: No Python libraries must be installed using apt-get - this will lead to errors using pip3
sudo apt-get -y install python3-pip
sudo pip install --ignore-installed --upgrade pip
# Remove Python and system packages that TorBox no longer uses (keeps the image small).
# The old TCS/TFS (Flask) and the CAPTCHA solver of bridges_get.py needed them.
for PACKAGE in pipenv virtualenv aniso8601 bidict blinker eventlet flask flask-compress flask-cors flask-restful flask-socketio greenlet gunicorn itsdangerous peewee pgpy python-engineio python-socketio simple-websocket standard-imghdr werkzeug wsproto numpy opencv-python-headless pillow pytesseract; do
	(sudo pip3 uninstall -y $PACKAGE) &>/dev/null
done
sudo apt-get -y purge tesseract-ocr imagemagick libjpeg-dev
sudo apt-get -y autoremove
(sudo rm Pipfile.lock*) 2>/dev/null
(sudo rm requirements.txt) 2>/dev/null
# Download requirements.txt (generated by the maintainers from Pipfile.lock with "pipenv requirements")
wget --no-cache -O requirements.txt https://raw.githubusercontent.com/$TORBOXMENU_FORKNAME/TorBox/$TORBOXMENU_BRANCHNAME/requirements.txt
# NEW v.0.5.5: Test if requirements.txt is able to install these packages bellow
sudo sed -i "/^pip==.*/d" requirements.txt
sudo sed -i "s/^typing-extensions==/typing_extensions==/g" requirements.txt
if [ -f "requirements.failed" ]; then rm requirements.failed; fi
clear
echo -e "${YELLOW}The following Python modules are installed:${NOCOLOR}"
REPLY="Y"
while [ "$REPLY" == "Y" ] || [ "$REPLY" == "y" ]; do
	REPLY=""
	# NEW v.0.5.4
	# grep -v '^\s*$ filters out empty lines or lines containing only whitespace.
	# tail -n +2 will skipp the first line
	readarray -t REQUIREMENTS < <(grep -v '^\s*$' requirements.txt | tail -n +2)
  for REQUIREMENT in "${REQUIREMENTS[@]}"; do
		# NEW v.0.5.4
		if grep "==" <<< $REQUIREMENT ; then REQUIREMENT=$(sed s"/==.*//" <<< $REQUIREMENT); fi
		VERSION=$(pip3 freeze | grep -i $REQUIREMENT== | sed "s/${REQUIREMENT}==//i" 2>&1)
    echo -e "${RED}${REQUIREMENT} version: ${YELLOW}$VERSION${NOCOLOR}"
    if [ -z "$VERSION" ]; then
      # shellcheck disable=SC2059
      (printf "$REQUIREMENT\n" | tee -a requirements.failed) >/dev/null 2>&1
    fi
  done
  if [ -f requirements.failed ]; then
    echo ""
    echo -e "${YELLOW}Not all required Python modules could be installed!${NOCOLOR}"
    read -r -p $'\e[1;93mWould you like to try it again [Y/n]? -> \e[0m'
    if [[ $REPLY =~ ^[YyNn]$ ]] ; then
      if [ "$REPLY" == "Y" ] || [ "$REPLY" == "y" ]; then
				# sudo pip3 install --ignore-installed -r requirements.txt
        sudo pip3 install --ignore-installed -r requirements.failed
        sleep 5
        rm requirements.failed
        unset REQUIREMENTS
        clear
      fi
    fi
  fi
done
cd torbox
unset REPLY
echo ""
read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'

# The additional network drivers are not installed on a TorBox mini or TorBox on a Cloud installation
if [ "$TORBOX_MINI" -eq "0" ] && [ "$ON_A_CLOUD" -eq "0" ]; then
  clear
  echo -e "${YELLOW}The following additional network drivers are installed:${NOCOLOR}"
  dkms status
	echo ""
  echo -e "${RED}Does it look right?${NOCOLOR}"
	read -r -p $'\e[1;93mWould you like to (re-)install the aditional network drivers [y/N]? -> \e[0m'
  if [[ $REPLY =~ ^[YyNn]$ ]] ; then
    if [ "$REPLY" == "Y" ] || [ "$REPLY" == "y" ]; then
			sudo dkms remove --all -q
			sudo bash install/install_network_drivers
    fi
  fi
  echo ""
  read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
  unset REPLY
fi
clear
echo -e "${YELLOW}[!] PREPARATIONS FOR THE IMAGE${NOCOLOR}"
echo -e "${RED}[+] Setting the correct time${NOCOLOR}"
printf "Setting the correct time zone: "
if [ -f "/etc/timezone" ]; then
	sudo mv /etc/timezone /etc/timezone.bak
	(printf "Etc/UTC" | sudo tee /etc/timezone) 2>&1
  printf " "
fi
sudo timedatectl set-timezone UTC
echo ""
echo ""
settime
clear
echo -e "${YELLOW}[!] PREPARATIONS FOR THE IMAGE${NOCOLOR}"
echo
echo -e "${RED}[+] Deactivating TorBox's automatic counteractions feature...${NOCOLOR}"
sudo pkill -f "log_check.py"
echo ""
echo -e "${RED}[+] Stopping and masking tor${NOCOLOR}"
sudo systemctl stop tor
sudo systemctl mask tor
sudo systemctl mask tor@default.service
echo ""
echo -e "${RED}[+] Deactivating all bridges${NOCOLOR}"
deactivate_obfs4_bridges NORESTART
sudo sed -i "s/^ClientTransportPlugin snowflake /#ClientTransportPlugin snowflake /g" ${TORRC}
sudo sed -i "s/^Bridge snowflake /#Bridge snowflake /g" ${TORRC}
sudo sed -i "s/^Bridge meek_lite /#Bridge meek_lite /g" ${TORRC}
echo ""
echo -e "${RED}[+] Deactivating the bridge relay${NOCOLOR}"
deactivating_bridge_relay
echo ""
echo -e "${RED}[+] Removing permanently OBFS4 Bridge Relay data${NOCOLOR}"
(sudo rm -r /var/lib/tor/keys) 2>/dev/null
(sudo rm /var/lib/tor/fingerprint) 2>/dev/null
(sudo rm /var/lib/tor/hashed-fingerprint) 2>/dev/null
(sudo rm -r /var/lib/tor/pt_state) 2>/dev/null
echo ""
echo -e "${RED}[+] Resetting Tor and force a change of the permanent entry node ${NOCOLOR}"
(sudo rm -r /var/lib/tor/cached-certs) 2>/dev/null
(sudo rm -r /var/lib/tor/cached-consensus) 2>/dev/null
(sudo rm -r /var/lib/tor/cached-descriptors) 2>/dev/null
(sudo rm -r /var/lib/tor/cached-descriptors.new) 2>/dev/null
(sudo rm -r /var/lib/tor/cached-microdesc-consensus) 2>/dev/null
(sudo rm -r /var/lib/tor/cached-microdescs) 2>/dev/null
(sudo rm -r /var/lib/tor/cached-microdescs.new) 2>/dev/null
(sudo rm -r /var/lib/tor/diff-cache) 2>/dev/null
(sudo rm -r /var/lib/tor/lock) 2>/dev/null
(sudo rm -r /var/lib/tor/state) 2>/dev/null
echo ""
echo -e "${RED}[+] Deleting all stored wireless passwords${NOCOLOR}"
(sudo rm /etc/wpa_supplicant/wpa_supplicant.conf) 2>/dev/null
(sudo rm /etc/wpa_supplicant/wpa_supplicant-wlan0.conf) 2>/dev/null
(sudo rm /etc/wpa_supplicant/wpa_supplicant-wlan1.conf) 2>/dev/null
echo ""
echo -e "${RED}[+] Copy default iptables.ipv4.nat${NOCOLOR}"
if [ "$TORBOX_MINI" -eq "1" ]; then
	sudo cp $TORBOX_PATH/etc/iptables.ipv4-mini.nat /etc/iptables.ipv4.nat
else
  sudo cp $TORBOX_PATH/etc/iptables.ipv4.nat /etc/
fi
if [ "$ON_A_CLOUD" -eq "0" ]; then
	echo ""
	echo -e "${RED}[+] Copy default interfaces${NOCOLOR}"
	if [ "$TORBOX_MINI" -eq "1" ]; then
		sudo cp $TORBOX_PATH/etc/network/interfaces.mini /etc/network/interfaces
	else
  	sudo cp $TORBOX_PATH/etc/network/interfaces /etc/network/
	fi
fi
echo ""
echo -e "${RED}[+] Erasing big not usefull packages...${NOCOLOR}"
# Find the bigest space waster packages: dpigs -H
(sudo apt-get -y --purge remove libgl1-mesa-dri texlive* lmodern) 2>/dev/null
echo ""
echo -e "${RED}[+] Fixing and cleaning${NOCOLOR}"
sudo apt --fix-broken install
sudo apt-get -y clean; sudo apt-get -y autoclean; sudo apt-get -y autoremove
go clean -cache
echo ""
echo -e "${RED}[+] Removing APT cache completely to reduce image size...${NOCOLOR}"
sudo rm -rf /var/cache/apt/archives/*
sudo rm -rf /var/lib/apt/lists/*
echo ""
echo -e "${RED}[+] Removing Python pip cache to reduce image size...${NOCOLOR}"
sudo rm -rf /root/.cache/pip
sudo rm -rf /home/torbox/.cache/pip
sudo rm -rf /home/torbox/.local/share/pip
echo ""
echo -e "${RED}[+] Removing Go build cache to reduce image size...${NOCOLOR}"
sudo rm -rf /root/.cache/go-build
sudo rm -rf /home/torbox/.cache/go-build
sudo rm -rf /root/go
sudo rm -rf /home/torbox/go
echo ""
echo -e "${RED}[+] Removing all user cache directories to reduce image size...${NOCOLOR}"
sudo rm -rf /home/torbox/.cache/*
sudo rm -rf /root/.cache/*
echo ""
echo -e "${RED}[+] Removing temporary files to reduce image size...${NOCOLOR}"
sudo rm -rf /tmp/*
sudo rm -rf /var/tmp/*
echo ""
echo -e "${RED}[+] Reducing systemd journal size to minimum...${NOCOLOR}"
sudo journalctl --vacuum-size=1M
sudo journalctl --vacuum-time=1d
echo ""
echo -e "${RED}[+] Removing bash history files...${NOCOLOR}"
(rm /home/torbox/.bash_history) 2>/dev/null
(sudo rm /root/.bash_history) 2>/dev/null
sudo setcap 'cap_net_bind_service=+ep' /usr/bin/obfs4proxy
sudo sed -i "s/^NoNewPrivileges=yes/NoNewPrivileges=no/g" /lib/systemd/system/tor@default.service
sudo sed -i "s/^NoNewPrivileges=yes/NoNewPrivileges=no/g" /lib/systemd/system/tor@.service
sudo systemctl daemon-reload
echo ""
echo -e "${RED}[+] Setting log level to low...${NOCOLOR}"
LOG_STATUS_001=$(sudo systemctl is-active rsyslog)
LOG_STATUS_002=$(sudo systemctl is-active systemd-journald.service)
if [ "$LOG_STATUS_001" != "inactive" ] || [ "$LOG_STATUS_002" != "inactive" ]; then
	echo ""
  echo -e "${RED}[+] Stopping logging now...${NOCOLOR}"
  sudo systemctl stop rsyslog
  sudo systemctl stop systemd-journald-dev-log.socket
  sudo systemctl stop systemd-journald-audit.socket
  sudo systemctl stop systemd-journald.socket
  sudo systemctl stop systemd-journald.service
  sudo systemctl mask systemd-journald.service
  echo ""
  echo -e "${RED}[+] Making it permanent...${NOCOLOR}"
  #Siehe auch hier: https://stackoverflow.com/questions/17358499/linux-how-to-disable-all-log
  sudo systemctl disable rsyslog
  sudo systemctl mask rsyslog
fi
echo ""
echo -e "${RED}[+] Deleting all logs and resetting Tor statistics...${NOCOLOR}"
echo
erase_logs
echo ""
echo -e "${RED}[+] Setting the right start trigger${NOCOLOR}"
sudo sed -i "s/^FRESH_INSTALLED=.*/FRESH_INSTALLED=2/" ${RUNFILE}
echo ""
echo -e "${YELLOW}[!] PREPARATIONS FOR THE IMAGE IS FINISHED!${NOCOLOR}"
echo -e "${RED}[+] We will shutdown the TorBox now.${NOCOLOR}"
echo ""
read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
clear
sudo shutdown -h now
