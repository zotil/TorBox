#!/bin/bash
# shellcheck disable=SC2001,SC2004,SC2181

# This file is a part of TorBox, an easy to use anonymizing router based on Raspberry Pi.
# Copyright (C) 2026 radio_24
# Contact: anonym@torbox.ch
# Website: https://www.torbox.ch
# Github:  https://github.com/radio24/TorBox
#
# Copyright (C) 2022 nyxnor (Contributor)
# Github:  https://github.com/nyxnor
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
# This script installs the newest version of TorBox on a clean, running
# Raspberry Pi OS lite. Please before starting the installation ensure that
# the user account "torbox" is already created and that you are logged in as such.
#
# SYNTAX
# ./run_install.sh [-h|--help] [--randomize_hostname] [--select-tor] [--select-fork fork_owner_name] [--select-branch branch_name] [--on_a_cloud] [--torbox_mini] [--step_by_step] [--continue_with_step]
#
# The -h or --help option shows the help screen.
#
# The --randomize_hostname option is helpful for people in highly authoritarian
# countries to avoid their ISP seeing their default hostname. The ISP can
# see and even block your hostname. When a computer connects to an ISP's
# network, it sends a DHCP request that includes the hostname.
#
# The --select-tor option allows to select a specific tor version. Without
# this option, the installation script installs the latest stable version.
#
# The --select-fork option allows to install a specific fork. The
# fork_owner_name is the GitHub user name of the for-owner.
#
# The --select-branch option allows to install a specific TorBox branch.
# Without this option, the installation script installs the master branch.
#
# The --on_a_cloud option has to be used if you install TorBox on a cloud or
# as a cloud service. This will enable/disable some features.
#
# The --torbox_mini option creates the TorBox mini on a Raspberry Pi Zero 2 W.
# Important: # Before the script can be started with this option, the Raspberry
# Pi OS lite 32-bit must be installed on the SD card running in the Raspberry Pi Zero 2 W.
#
# The --step_by_step option execute the installation step by step, which
# is ideal to find bugs.
#
# The --continue_with_step In case of an aborted installation, this option
# allows to continue the installation with a certain step, skipping all other
# steps before.
#
# IMPORTANT
# Start the insatllation as user "torbox""!
# Dont run it as root (no sudo)!
#
##########################################################

# Table of contents for this script:
#  0.  Checking for Internet connection
#  1.  Adjusting time, if needed
#  2a. Checking for the WLAN regulatory domain
#  2b. Removing Networkmanager and ModemManager
#  3.  Updating the system
#  4.  Installing all necessary packages
#  5.  Install Tor
#  6.  Configuring Tor with its pluggable transports
#  7.  Install Snowflake
#  8.  Re-checking Internet connectivity
#  9.  Downloading and installing the latest version of TorBox
# 10.  Installing all configuration files
# 11.  Disabling Bluetooth
# 12.  Configure the system services
# 13.  Updating run/torbox.run
# 14.  TorBox mini specific configurations
# 15.  Finishing, cleaning and booting

##########################################################

##### SET VARIABLES ######
#
# Set the the variables for the menu
MENU_WIDTH=80
MENU_WIDTH_REDUX=60
MENU_HEIGHT_25=25
MENU_HEIGHT_20=20
MENU_HEIGHT_10=10

# Colors
RED='\033[1;31m'
YELLOW='\033[1;93m'
NOCOLOR='\033[0m'

# What main version is installed
DEBIAN_VERSION=$(sed 's/\..*//' /etc/debian_version)

# Where is the config.txt?
if [ "$DEBIAN_VERSION" -gt "11" ]; then
  CONFIGFILE="/boot/firmware/config.txt"
else
  CONFIGFILE="/boot/config.txt"
fi

# Where is the cmdline.txt?
if [ "$DEBIAN_VERSION" -gt "11" ]; then
  CMDLINEFILE="/boot/firmware/cmdline.txt"
else
  CMDLINEFILE="/boot/cmdline.txt"
fi

# Changes in the variables below (until the ####### delimiter) will be saved
# into run/torbox.run and used after the installation (we not recommend to
# change the values until zou precisely know what you are doing)
# Public nameserver used to circumvent cheap censorship
NAMESERVERS="1.1.1.1,1.0.0.1,8.8.8.8,8.8.4.4"

# Default hostname
HOSTNAME="TorBox055"

# For go
GO_DL_PATH="https://go.dev/dl/"
GO_PROGRAM="/usr/local/go/bin/go"

# Release Page of the official Tor repositories
TORURL="https://gitlab.torproject.org/tpo/core/tor/-/tags"
TORPATH_TO_RELEASE_TAGS="/tpo/core/tor/-/tags/tor-"
# WARNING: Sometimes, GitLab will change this prefix! With .* use sed -e
TOR_HREF_FOR_SED="<a class=\".*\" href=\"/tpo/core/tor/-/tags/tor-"
TOR_HREF_FOR_SED_NEW="<a href=\"/tpo/core/tor/-/tags/tor-"
TORURL_DL_PARTIAL="https://dist.torproject.org/tor-"

# Snowflake repositories
SNOWFLAKE_ORIGINAL_WEB="https://gitlab.torproject.org/tpo/anti-censorship/pluggable-transports/snowflake"
# Only until version 2.6.1
SNOWFLAKE_PREVIOUS_USED="https://github.com/syphyr/snowflake"
# Version 2.8.0+
SNOWFLAKE_USED="https://github.com/tgragnato/snowflake"

# OBFS4Proxy
OBFS4PROXY_USED="https://salsa.debian.org/pkg-privacy-team/obfs4proxy.git"

# Connectivity check
CHECK_URL1="debian.org"
CHECK_URL2="google.com"

# Catching command line options
OPTIONS=$(getopt -o h --long help,randomize_hostname,select-tor,select-fork:,select-branch:,on_a_cloud,torbox_mini,step_by_step,continue_with_step: -n 'run-install' -- "$@")
if [ $? != 0 ] ; then echo "Syntax error!"; echo ""; OPTIONS="-h" ; fi
eval set -- "$OPTIONS"

SELECT_TOR=
SELECT_BRANCH=
TORBOXMENU_BRANCHNAME=
TORBOXMENU_FORKNAME=
ON_A_CLOUD=
STEP_BY_STEP=
while true; do
  case "$1" in
    -h | --help )
			echo "Copyright (C) 2026 radio_24, nyxnor (Contributor)"
			echo "Syntax : run_install.sh [-h|--help] [--randomize_hostname] [--select-tor] [--select-fork fork_name] [--select-branch branch_name] [--on_a_cloud] [--torbox_mini] [--step_by_step] [--continue_with_step]"
			echo "Options: -h, --help     : Shows this help screen ;-)"
			echo "         --randomize_hostname"
			echo "                        : Randomizes the hostname to prevent ISPs to see the default"
			echo "         --select-tor   : Let select a specific tor version (default: newest stable version)"
			echo "         --select-fork fork_owner_name"
			echo "                        : Let select a specific fork from a GitHub user (fork_owner_name)"
			echo "         --select-branch branch_name"
			echo "                        : Let select a specific TorBox branch (default: master)"
			echo "         --on_a_cloud   : Installing on a cloud or as a cloud service"
      echo "         --torbox_mini  : Installing TorBox mini on a Raspberry Pi Zero 2 W"
			echo "         --step_by_step : Executes the installation step by step"
      echo "         --continue_with_step"
      echo "                        : Continue the installation with a certain step"
			echo ""
			echo "Please before starting the installation ensure that the user account \"torbox\" is already created"
			echo "and that you are logged in as such."
			echo "For more information visit https://www.torbox.ch/ or https://github.com/radio24/TorBox"
			exit 0
	  ;;
		--randomize_hostname ) RANDOMIZE_HOSTNAME=1; shift ;;
    --select-tor ) SELECT_TOR="--select-tor"; shift ;;
		--select-fork )
		  # shellcheck disable=SC2034
			SELECT_FORK="--select-fork"
			[ ! -z "$2" ] && TORBOXMENU_FORKNAME="$2"
			shift 2
		;;
    --select-branch )
		  # shellcheck disable=SC2034
			SELECT_BRANCH="--select-branch"
			[ ! -z "$2" ] && TORBOXMENU_BRANCHNAME="$2"
			shift 2
		;;
		--on_a_cloud ) ON_A_CLOUD="--on_a_cloud"; shift ;;
    --torbox_mini ) TORBOX_MINI="--torbox_mini"; shift ;;
    --step_by_step ) STEP_BY_STEP="--step_by_step"; shift ;;
    --continue_with_step )
      # shellcheck disable=SC2034
      CONTINUE_WITH_STEP="--continue_with_step"
      [ ! -z "$2" ] && STEP_NUMBER="$2"
      shift 2
    ;;
		-- ) shift; break ;;
		* ) break ;;
  esac
done

[ -z "$STEP_NUMBER" ] && STEP_NUMBER="1"

# TorBox Repository
[ -z "$TORBOXMENU_FORKNAME" ] && TORBOXMENU_FORKNAME="radio24"
[ -z "$TORBOXMENU_BRANCHNAME" ] && TORBOXMENU_BRANCHNAME="master"
TORBOXURL="https://github.com/$TORBOXMENU_FORKNAME/TorBox/archive/refs/heads/$TORBOXMENU_BRANCHNAME.zip"

#Other variables
RUNFILE="torbox/run/torbox.run"
i=0
n=0

######## PREPARATIONS ########
#
# Configure variable for resolv.conf, if needed
NAMESERVERS_ORIG=$NAMESERVERS
ONE_NAMESERVER=$(cut -d ',' -f1 <<< $NAMESERVERS)
NAMESERVERS=$(cut -f2- -d ',' <<< $NAMESERVERS)
i=0
while [ "$ONE_NAMESERVER" != " " ]
do
	if [ $i = 0 ]; then
		RESOLVCONF="\n# Added by TorBox install script\n"
	fi
	RESOLVCONF="${RESOLVCONF}nameserver $ONE_NAMESERVER\n"
	i=$((i+1))
	if [ "$ONE_NAMESERVER" = "$NAMESERVERS" ]; then
		ONE_NAMESERVER=" "
	else
		ONE_NAMESERVER=$(cut -d ',' -f1 <<< $NAMESERVERS)
		NAMESERVERS=$(cut -f2- -d ',' <<< $NAMESERVERS)
	fi
done

##############################
######## FUNCTIONS ###########

# New function re-connect
# This function tries to restor a connection to the Internet after failing to install a package
# Syntax: re-connect()
re-connect()
{
	if [ -f "/etc/resolv.conf" ]; then
		(sudo cp /etc/resolv.conf /etc/resolv.conf.bak) 2>&1
	fi
	(sudo printf "$RESOLVCONF" | sudo tee /etc/resolv.conf) 2>&1
	sleep 5
	ping -c 1 -q $CHECK_URL1 >&/dev/null
	OCHECK=$?
	echo ""
	if [ $OCHECK -eq 0 ]; then
	  echo -e "${RED}[+]         Yes, we have Internet! :-)${NOCOLOR}"
	else
	  echo -e "${YELLOW}[!]        Hmmm, no we don't have Internet... :-(${NOCOLOR}"
	  echo -e "${RED}[+]        We will check again in about 30 seconds...${NOCOLOR}"
	  sleep 30
	  echo ""
	  echo -e "${RED}[+]         Trying again...${NOCOLOR}"
	  ping -c 1 -q $CHECK_URL2 >&/dev/null
	  if [ $? -eq 0 ]; then
	    echo -e "${RED}[+]         Yes, now, we have an Internet connection! :-)${NOCOLOR}"
	  else
	    echo -e "${YELLOW}[!]         Hmmm, still no Internet connection... :-(${NOCOLOR}"
	    echo -e "${RED}[+]         We will try to catch a dynamic IP adress and check again in about 30 seconds...${NOCOLOR}"
			sudo dhcpcd -n
	    echo ""
	    echo -e "${RED}[+]         Trying again...${NOCOLOR}"
	    ping -c 1 -q $CHECK_URL1 >&/dev/null
	    if [ $? -eq 0 ]; then
	      echo -e "${RED}[+]         Yes, now, we have an Internet connection! :-)${NOCOLOR}"
	    else
				echo -e "${RED}[+]         Hmmm, still no Internet connection... :-(${NOCOLOR}"
				echo -e "${RED}[+]         Internet connection is mandatory. We cannot continue - giving up!${NOCOLOR}"
				echo -e "${RED}[+]         Please, try to fix the problem and re-run the installation!${NOCOLOR}"
				exit 1
	    fi
	  fi
	fi
}

# Modified to check, if the packages was installed
# This function installs the packages in a controlled way, so that the correct
# installation can be checked.
# Syntax check_install_packages <packagenames>
check_install_packages()
{
 packagenames=$1
 for packagename in $packagenames; do
	 check_installed=0
	 while [ $check_installed == "0" ]; do
	 	clear
		echo -e "${RED}[+] Step 4: Installing all necessary packages....${NOCOLOR}"
	 	echo ""
	 	echo -e "${RED}[+]         Installing ${YELLOW}$packagename${NOCOLOR}"
	 	echo ""
	 	sudo apt-get -y install $packagename
		check=$(dpkg-query -s $packagename | grep "Status" | grep -o "installed")
		if [ "$check" == "installed" ]; then check_installed=1
		else re-connect
		fi
	done
 done
}

# This function downloads and updates tor
# Syntax download_and_compile_tor
# Used predefined variables: $download_tor_url $filename RED WHITE NOCOLOR TORCONNECT
download_and_compile_tor()
{
	# Difference to the update-function - we cannot use torsocks yet
	wget $download_tor_url
	DLCHECK=$?
	if [ $DLCHECK -eq 0 ]; then
		echo -e "${RED}[+]         Successfully downloaded the selected tor version... ${NOCOLOR}"
		tar xzf $filename
		cd $directoryname
		echo -e "${RED}[+]         Starting configuring, compiling and installing... ${NOCOLOR}"
		# Give it a touch of git (without these lines the compilation will break with a git error)
		git init
		git add -- *
		git config --global user.name "torbox"
		git config --global user.email "torbox@localhost"
		git commit -m "Initial commit"
		# Don't use ./autogen.sh
		if [ -f "autogen.sh" ]; then sh autogen.sh; fi
		sh configure --disable-unittests
		make
		sudo make install
		cd
		sudo rm -r tor-*
		sudo systemctl stop tor
		sudo systemctl mask tor
		# Both tor services have to be masked to block outgoing tor connections
		sudo systemctl mask tor@default.service
		sudo systemctl stop tor
		sudo systemctl mask tor
		# Both tor services have to be masked to block outgoing tor connections
		sudo systemctl mask tor@default.service
	else
		echo -e ""
		echo -e "${YELLOW}[!] COULDN'T DOWNLOAD TOR!${NOCOLOR}"
		echo -e "${RED}[+] The official Tor repositories may be blocked or offline!${NOCOLOR}"
		echo -e "${RED}[+] Please try again later and if the problem persists, please report it${NOCOLOR}"
		echo -e "${RED}[+] to ${YELLOW}anonym@torbox.ch${RED}. ${NOCOLOR}"
		echo ""
		read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
		clear
		exit 0
	fi
}

# select_and_install_tor()
# Syntax select_and_install_tor
# Used predefined variables: RED, WHITE, NOCOLOR, SELECT_TOR, TORURL, TORPATH_TO_RELEASE_TAGS, TOR_HREF_FOR_SED_NEW
# With this function change/update of tor from a list of versions is possible
# IMPORTANT: This function is different from the one in the update script!
select_and_install_tor()
{
  # Difference to the update-function - we cannot use torsocks yet
	echo -e "${RED}[+]         Can we access the official Tor repositories on GitHub?${NOCOLOR}"
	#-m 6 must not be lower, otherwise it looks like there is no connection! ALSO IMPORTANT: THIS WILL NOT WORK WITH A CAPTCHA!
	OCHECK=$(curl -m 6 -s $TORURL)
	if [ $? == 0 ]; then
		echo -e "${YELLOW}[!]         YES!${NOCOLOR}"
		echo ""
	else
		echo -e "${YELLOW}[!]         NO!${NOCOLOR}"
		echo -e ""
		echo -e "${RED}[+] The official Tor repositories may be blocked or offline!${NOCOLOR}"
		echo -e "${RED}[+] Please try again later and if the problem persists, please report it${NOCOLOR}"
		echo -e "${RED}[+] to ${YELLOW}anonym@torbox.ch${RED}. ${NOCOLOR}"
		echo ""
		echo -e "${RED}[+] However, an older version of tor is already installed from${NOCOLOR}"
		echo -e "${RED}    the Raspberry PI OS repository.${NOCOLOR}"
		read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
		clear
	fi
	echo -e "${RED}[+]         Fetching possible tor versions... ${NOCOLOR}"
	# With TOR_HREF_FOR_SED, because of .*, sed -e has to be used!!
	readarray -t torversion_versionsorted < <(curl --silent $TORURL | grep $TORPATH_TO_RELEASE_TAGS | sed "s|$TOR_HREF_FOR_SED_NEW||g" | sed -e "s/\">.*//g" | sed -e "s/ //g" | sort -r)


  #How many tor version did we fetch?
	number_torversion=${#torversion_versionsorted[*]}
	if [ $number_torversion = 0 ]; then
		echo -e ""
		echo -e "${YELLOW}[!] COULDN'T FIND ANY TOR VERSIONS${NOCOLOR}"
		echo -e "${RED}[+] The official Tor repositories may be blocked or offline!${NOCOLOR}"
		echo -e "${RED}[+] Please try again later and if the problem persists, please report it${NOCOLOR}"
		echo -e "${RED}[+] to ${YELLOW}anonym@torbox.ch${RED}. ${NOCOLOR}"
		echo ""
		echo -e "${RED}[+] However, an older version of tor is already installed from${NOCOLOR}"
		echo -e "${RED}    the Raspberry PI OS repository.${NOCOLOR}"
		echo ""
		read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
		clear
  else
		#We will build a new array with only the relevant tor versions
    i=0
    while [ $i -lt $number_torversion ]
    do
      if [ $n = 0 ]; then
        torversion_versionsorted_new[0]=${torversion_versionsorted[0]}
        covered_version_full=${torversion_versionsorted[0]}
        covered_version=$(cut -d '.' -f1-3 <<< ${torversion_versionsorted[0]})
        i=$((i+1))
        n=$((n+1))
      else
        actual_version_full=${torversion_versionsorted[$i]}
        actual_version=$(cut -d '.' -f1-3 <<< ${torversion_versionsorted[$i]})
        if [ "$actual_version" == "$covered_version" ]; then
          covered_version_work="$(<<< "$covered_version_full" sed -e 's/\.//g' | sed -e s/"\^{}\|\-[a-z].*$"//g)"
          actual_version_work="$(<<< "$actual_version_full" sed -e 's/\.//g' | sed -e s/"\^{}\|\-[a-z].*$"//g)"
          if [ $actual_version_work -le $covered_version_work ]; then i=$((i+1))
          else
            n=$((n-1))
            torversion_versionsorted_new[$n]=${torversion_versionsorted[$i]}
            covered_version_full=$actual_version_full
            covered_version=$actual_version
            i=$((i+1))
            n=$((n+1))
          fi
        else
          torversion_versionsorted_new[$n]=${torversion_versionsorted[$i]}
          covered_version_full=$actual_version_full
          covered_version=$actual_version
          i=$((i+1))
          n=$((n+1))
        fi
      fi
    done
    number_torversion=$n

    #Display and chose a tor version
		if [ "$SELECT_TOR" = "--select-tor" ]; then
			clear
			echo -e "${YELLOW}Choose a tor version (alpha versions are not recommended!):${NOCOLOR}"
    	echo ""
    	for (( i=0; i<$number_torversion; i++ ))
    	do
      	menuitem=$((i+1))
      	echo -e "${RED}$menuitem${NOCOLOR} - ${torversion_versionsorted_new[$i]}"
    	done
    	echo ""
    	read -r -p $'\e[1;93mWhich tor version (number) would you like to use? -> \e[0m'
    	echo
    	if [[ $REPLY =~ ^[1234567890]$ ]]; then
				if [ $REPLY -gt 0 ] && [ $((REPLY-1)) -le $number_torversion ]; then
        	CHOICE_TOR=$((REPLY-1))
        	clear
        	echo -e "${RED}[+]         Download the selected tor version...... ${NOCOLOR}"
        	version_string="$(<<< ${torversion_versionsorted_new[$CHOICE_TOR]} sed -e 's/ //g')"
        	download_tor_url="$TORURL_DL_PARTIAL$version_string.tar.gz"
        	filename="tor-$version_string.tar.gz"
					directoryname="tor-$version_string"
					download_and_compile_tor
				else
					clear
					echo -e "${YELLOW}[!] WRONG SELECTION!${NOCOLOR}"
	       	echo -e "${RED}[+] Restart the installation and try again! ${NOCOLOR}"
					echo ""
					sleep 5
					clear
					exit 0
				fi
    	else
				clear
				echo -e "${YELLOW}[!] WRONG SELECTION!${NOCOLOR}"
				echo -e "${RED}[+] Restart the installation and try again! ${NOCOLOR}"
				echo ""
				sleep 5
				clear
				exit 0
			fi

		#Install the latest stable version of tor
		else
			echo ""
			echo -e "${RED}[+]         Selecting a tor version to install.${NOCOLOR}"
    	for (( i=0; i<$number_torversion; i++ ))
    	do
				if grep -v "-" <<< "${torversion_versionsorted_new[$i]}"; then
					version_string="$(<<< ${torversion_versionsorted_new[$i]} sed -e 's/ //g')"
					download_tor_url="$TORURL_DL_PARTIAL$version_string.tar.gz"
        	filename="tor-$version_string.tar.gz"
					directoryname="tor-$version_string"
					i=$number_torversion
				fi
    	done
			echo ""
			echo -e "${RED}[+]         Selected tor version ${YELLOW}$version_string${RED}...${NOCOLOR}"
			echo -e "${RED}[+]         Download the selected tor version...... ${NOCOLOR}"
			download_and_compile_tor
		fi
	fi
}

###### DISPLAY THE INTRO ######
clear
if (whiptail --title "TorBox Installation on Raspberry Pi OS (scroll down!)" --scrolltext --no-button "STOP!" --yes-button "INSTALL" --yesno "         WELCOME TO THE INSTALLATION OF TORBOX ON RASPBERRY PI OS\n\nBefore we start, please ensure that you have already created a user account \"torbox\" and are currently logged in as such. Also, at the end of the installation, we will remove Rasperi Pi OS's auto-login feature - be sure you know your password for \"torbox\"!!\n\nBy the way, this script should be started as \"./run_install\" (without sudo !!) in your home directory, which is \"/home/torbox\".The installation process runs almost without user interaction. However, macchanger will ask for enabling an autmatic change of the MAC address - REPLY WITH NO!\n\nTHIS INSTALLATION WILL CHANGE/DELETE THE CURRENT CONFIGURATION!\n\nIMPORTANT\nInternet connectivity is necessary for the installation.\n\nAVAILABLE OPTIONS\n-h, --help     : shows a help screen\n--randomize_hostname\n  	  	   : randomizes the hostname to prevent ISPs to see the default\n--select-tor   : select a specific tor version\n--select-fork fork_owner_name\n  	  	   : select a specific fork from a GitHub user\n--select-branch branch_name\n  	  	   : select a specific TorBox branch\n--on_a_cloud   : installing on a cloud or as a cloud service.\n--torbox_mini  : installing TorBox mini on a Raspberry Pi Zero 2 W.\n--step_by_step : executes the installation step by step.\n--continue_with_step\n  	  	   : continue the installation with a certain step.\n\nIn case of any problems, contact us on https://www.torbox.ch." $MENU_HEIGHT_25 $MENU_WIDTH); then
	:
else
	clear
	exit
fi

if [ -z "$RANDOMIZE_HOSTNAME" ]; then
	if (whiptail --title "TorBox Installation on Raspberry Pi OS" --defaultno --no-button "USE DEFAULT" --yes-button "CHANGE!" --yesno "In highly authoritarian countries connecting the tor network could be seen as suspicious. The default hostname of TorBox is \"TorBox<nnn>\" (<nnn> representing the version).\n\nWhen a computer connects to an ISP's network, it sends a DHCP request that includes the hostname. Because ISPs can see, log and even block hostnames, setting another hostname or using a randomized hostname may be preferable.\n\nWe recommend randomizing the hostname in highly authoritarian countries or if you think that your ISP blocks tor related network traffic.\n\nDo you want to use the DEFAULT hostname or to CHANGE it?" $MENU_HEIGHT_20 $MENU_WIDTH); then
		if (whiptail --title "TorBox Installation on Raspberry Pi OS" --no-button "SET HOSTNAME" --yes-button "RANDOMIZE HOSTNAME" --yesno "You can set a specific hostname or use a randomized one. Please choose..." $MENU_HEIGHT_10 $MENU_WIDTH); then
			# shellcheck disable=SC2002
			HOSTNAME=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 10 | head -n 1)
		else
			HOSTNAME=$(whiptail --title "TorBox Installation on Raspberry Pi OS" --inputbox "\nEnter the hostname:" $MENU_HEIGHT_10 $MENU_WIDTH_REDUX 3>&1 1>&2 2>&3)
			if [[ $HOSTNAME != *[0123456789ABCDEFGHIJKLMNOPQRSTUVWXZYabcdefghijklmnopqrstuvwxzy-]* ]]; then
				HOSTNAME=$(tr -dc 'a-zA-Z0-9' <<<$HOSTNAME)
			fi
			if ${#HOSTNAME} -gt 64 ; then
				HOSTNAME=$(head -c 64 <<<$HOSTNAME)
			fi
		fi
	fi
else
	# shellcheck disable=SC2002
	HOSTNAME=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 10 | head -n 1)
fi

# 0. Checking for Internet connection
clear
echo -e "${RED}[+] Step 0: Do we have Internet?${NOCOLOR}"
echo -e "${RED}[+]         Nevertheless, to be sure, let's add some open nameservers!${NOCOLOR}"
re-connect
# NEW v.0.5.4-post
# Avahi can lead loosing the IP address
# See here: https://www.heise.de/ratgeber/Raspi-mit-Debian-verliert-Internet-Verbindung-9998575.html
sudo systemctl mask avahi-daemon
sudo systemctl disable avahi-daemon
sudo systemctl stop avahi-daemon
sudo systemctl mask avahi-daemon.socket
sudo systemctl disable avahi-daemon.socket
sudo systemctl stop avahi-daemon.socket

if [ "$STEP_NUMBER" -le "1" ]; then
  # 1. Adjusting time, if needed
  clear
  if [ -f "/etc/timezone" ]; then
	  sudo mv /etc/timezone /etc/timezone.bak
	  (printf "Etc/UTC" | sudo tee /etc/timezone) 2>&1
  fi
  sudo timedatectl set-timezone UTC
  clear
  echo -e "${YELLOW}[!] SYSTEM-TIME CHECK${NOCOLOR}"
  echo -e "${RED}[!] Tor needs a correctly synchronized time.${NOCOLOR}"
  echo -e "${RED}    The system should display the current UTC time:${NOCOLOR}"
  echo
  echo -e "             Date: ${YELLOW}$(date '+%Y-%m-%d')${NOCOLOR}"
  echo -e "             Time: ${YELLOW}$(date '+%H:%M')${NOCOLOR}"
  echo
  echo -e "${RED}    You can find the correct time here: ${YELLOW}https://time.is/UTC${NOCOLOR}"
  echo
  while true
  do
	  read -r -p $'\e[1;31m    Do you want to adjust the system time [Y/n]? -> \e[0m'
	  # The following line is for the prompt to appear on a new line.
	  if [[ $REPLY =~ ^[YyNn]$ ]] ; then
		  echo
		  echo
		  break
	  fi
  done
  if [[ $REPLY =~ ^[Yy]$ ]] ; then
	  echo ""
	  read -r -p $'\e[1;31mPlease enter the date (YYYY-MM-DD): \e[0m' DATESTRING
	  echo ""
	  echo -e "${RED}Please enter the UTC time (HH:MM)${NOCOLOR}"
	  read -r -p $'You can find the correct time here: https://time.is/UTC: ' TIMESTRING
	  # Check and set date
    if [[ $DATESTRING =~ ^[1-2]{1}[0-9]{3}-[0-9]{2}-[0-9]{2}$ ]]; then
		  echo ""
		  sudo date -s "$DATESTRING"
		  echo -e "${RED}[+] Date set successfully!${NOCOLOR}"
		  if [[ $TIMESTRING =~ ^[0-9]{2}:[0-9]{2}$ ]]; then
			  echo ""
			  sudo date -s "$TIMESTRING"
			  echo -e "${RED}[+] Time set successfully!${NOCOLOR}"
		    sleep 5
			  clear
		  else
			  echo ""
			  echo -e "${YELLOW}[!] INVALIDE TIME FORMAT!${NOCOLOR}"
			  echo ""
			  read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
			  clear
		  fi
	  else
		  echo ""
		  echo -e "${YELLOW}[!] INVALIDE DATE FORMAT!${NOCOLOR}"
		  echo ""
		  read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
		  clear
	  fi
  fi
fi

if [ "$STEP_NUMBER" -le "2" ]; then
  # 2a. Check the status of the WLAN regulatory domain to be sure WiFi will work
  clear
  echo -e "${RED}[+] Step 2a: Check the status of the WLAN regulatory domain...${NOCOLOR}"
  COUNTRY=$(sudo iw reg get | grep country | cut -d " " -f2)
  if [ "$COUNTRY" = "00:" ]; then
    echo -e "${YELLOW}[!]          No WLAN regulatory domain set - that will lead to problems!${NOCOLOR}"
    echo -e "${YELLOW}[!]          Therefore we will set it to US! You can change it later.${NOCOLOR}"
    sudo iw reg set US
    INPUT="REGDOMAIN=US"
    sudo sed -i "s/^REGDOMAIN=.*/${INPUT}/" /etc/default/crda
  else
    echo -e "${RED}[+]          The WLAN regulatory domain is set correctly! ${NOCOLOR}"
  fi
  echo -e "${RED}[+]          To be sure we will unblock wlan, now! ${NOCOLOR}"
  sudo rfkill unblock wlan
  sleep 10

  # NEW v.0.5.4
  # 2b. Removing Networkmanager and ModemManager
  clear
  echo -e "${RED}[+] Step 2b: Removing NetworkManager and ModemManager${NOCOLOR}"
  sudo apt-get -y purge network-manager modemmanager
  sleep 10
fi

if [ "$STEP_NUMBER" -le "3" ]; then
  # 3. Updating the system
  clear
  echo -e "${RED}[+] Step 3: Updating the system...${NOCOLOR}"
	sudo apt-get -y update
  sudo apt-get -y dist-upgrade
  sudo apt-get -y clean
  sudo apt-get -y autoclean
  sudo apt-get -y autoremove

  if [ "$STEP_BY_STEP" = "--step_by_step" ]; then
	  echo ""
	  read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
	  clear
  else
	  sleep 10
  fi
fi

if [ "$STEP_NUMBER" -le "4" ]; then
  # 4. Installing all necessary packages
  clear
  echo -e "${RED}[+] Step 4: Installing all necessary packages....${NOCOLOR}"
	echo -e " "
	# NEW v.0.5.5: This is necessary for installing linux-headers, which replaces raspberrypi-kernel-headers on Debian Trixie
	LINUX_HEADERS="linux-headers-$(uname -r)"
  # Installation of standard packages
  if [ "$TORBOX_MINI" == "--torbox_mini" ]; then
		check_install_packages "hostapd isc-dhcp-client isc-dhcp-server usbmuxd dnsmasq bind9-dnsutils tcpdump iftop vnstat debian-goodies apt-transport-https dirmngr imagemagick tesseract-ocr ntpsec-ntpdate screen git openvpn ppp nyx apt-transport-tor qrencode nginx basez iptables ipset macchanger openssl ca-certificates lshw libjpeg-dev ifupdown"
  else
		check_install_packages "hostapd isc-dhcp-client isc-dhcp-server usbmuxd dnsmasq bind9-dnsutils tcpdump iftop vnstat debian-goodies apt-transport-https dirmngr imagemagick tesseract-ocr ntpsec-ntpdate screen git openvpn ppp nyx apt-transport-tor qrencode nginx basez iptables ipset macchanger openssl ca-certificates lshw $LINUX_HEADERS dkms libjpeg-dev ifupdown"
  fi
  # Installation of developer packages - THIS PACKAGES ARE NECESSARY FOR THE COMPILATION OF TOR!! Without them, tor will disconnect and restart every 5 minutes!!
  check_install_packages "build-essential automake libevent-dev libssl-dev asciidoc bc devscripts dh-apparmor libcap-dev liblzma-dev libsystemd-dev libzstd-dev quilt zlib1g-dev"
  # IMPORTANT tor-geoipdb installs also the tor package
  check_install_packages "tor-geoipdb"
  sudo systemctl stop tor
  sudo systemctl mask tor
  # Both tor services have to be masked to block outgoing tor connections
  sudo systemctl mask tor@default.service
  # An old version of easy-rsa was available by default in some openvpn packages
  if [[ -d /etc/openvpn/easy-rsa/ ]]; then
	  rm -rf /etc/openvpn/easy-rsa/
  fi

  if [ "$STEP_BY_STEP" = "--step_by_step" ]; then
	  echo ""
	  read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
	  clear
  fi

  # Additional installations for Python
  clear
  echo -e "${RED}[+] Step 4: Installing all necessary packages....${NOCOLOR}"
  echo ""
  echo -e "${RED}[+]         Installing ${YELLOW}Python modules${NOCOLOR}"
  echo ""

  # For RaspberryPi OS based on Debian Bookworm needed
  PYTHON_LIB_PATH=$(python3 -c "import sys; print(sys.path)" | cut -d ',' -f3 | sed "s/'//g" | sed "s/,//g" | sed "s/ //g")
  if [ -f "$PYTHON_LIB_PATH/EXTERNALLY-MANAGED" ] ; then
    sudo rm "$PYTHON_LIB_PATH/EXTERNALLY-MANAGED"
  fi

	# Install and check Python requirements
	# How to deal with Pipfile, Pipfile.lock and requirements.txt:
	# 1. Check the Pipfile --> is the package in the list?
	# 2. Execute: pipenv lock (this should only be done on a test system not during installation or to prepare an image!)
	# 3. Execute: pipenv requirements >requirements.txt
	# 4. Execute: sudo pip install -r requirements.txt (this will update outdated packages)
	# 5. Check the list of outdated packages: pip list --outdated
	# Remark: we install all Python libraries globally (as root) because otherwice some programs troubling to find the library in the local environment
	# NEW v.0.5.5: No Python libraries must be installed using apt-get - this will lead to errors using pip3
  cd
	check_install_packages "python3-pip"
	sudo pip install --ignore-installed --upgrade pip
	sudo pip3 install --ignore-installed pipenv
	# Download the latest Pipfile.lock
	wget --no-cache https://raw.githubusercontent.com/$TORBOXMENU_FORKNAME/TorBox/$TORBOXMENU_BRANCHNAME/Pipfile.lock
	pipenv requirements >requirements.txt
	# If the creation of requirements.txt failes then use the (most probably older) one from our repository
	#wget --no-cache https://raw.githubusercontent.com/$TORBOXMENU_FORKNAME/TorBox/$TORBOXMENU_BRANCHNAME/requirements.txt
	# NEW v.0.5.5: Test if requirements.txt is able to install these packages bellow
	sudo sed -i "/^pip==.*/d" requirements.txt
	sudo sed -i "s/^typing-extensions==/typing_extensions==/g" requirements.txt
	# re-connect (not needed, because of check_install_packages "python3-pip")
	# NEW v.0.5.5: Some older Python libraries are installed because of dependencies. This leads to problems with pip3. With --ignore-installed the old conflicting libraries will be overwritten.
	sudo pip3 install --ignore-installed -r requirements.txt
  sleep 5
  clear
  echo -e "${YELLOW}Following Python modules are installed:${NOCOLOR}"
	if [ -f "requirements.failed" ]; then rm requirements.failed; fi
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
		if [ -f "requirements.failed" ]; then
		  echo ""
		  echo -e "${YELLOW}Not all required Python modules could be installed!${NOCOLOR}"
		  read -r -p $'\e[1;93mWould you like to try it again [Y/n]? -> \e[0m'
		  if [[ $REPLY =~ ^[YyNn]$ ]] ; then
			  if [ "$REPLY" == "Y" ] || [ "$REPLY" == "y" ]; then
					sudo pip3 install --ignore-installed -r requirements.failed
				  sleep 5
				  rm requirements.failed
				  unset REQUIREMENTS
				  clear
			  fi
		  fi
	  fi
  done

  if [ "$STEP_BY_STEP" = "--step_by_step" ]; then
	  echo ""
	  read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
	  clear
  fi

  # Additional installation for go
  clear
  echo -e "${RED}[+] Step 4: Installing all necessary packages....${NOCOLOR}"
  echo ""
  echo -e "${RED}[+]         Installing ${YELLOW}go${NOCOLOR}"
  echo ""

	# NEW v.0.5.4: New way to download the current version of go (and we cover the case if TorBox mini is build on a Raspberry Pi 5 with a 32 bit version of Raspberry Pi OS)
	# This is for a 64bit Raspberry Pi OS
	if uname -m | grep -q -E "arm64|aarch64"; then PLATFORM="linux-arm64"
	# This is for a 64bit Intel compatable architecture as used on a cloud
	elif uname -m | grep -q -E "x86_64"; then PLATFORM="linux-amd64"
	# This is for a 32bit Raspberry Pi OS
	elif uname -m | grep -q -E "armv6l|armv7l"; then PLATFORM="linux-armv6l"
	else
		PLATFORM=""
		DLCHECK=1
	fi

  # Fetch the filename of the latest go version
	if [ "$PLATFORM" != "" ] ; then
  	GO_FILENAME=$(curl -s "$GO_DL_PATH" | grep "$PLATFORM" | grep -m 1 'class=\"download\"' | cut -d'"' -f6 | cut -d'/' -f3)
  	wget --no-cache "$GO_DL_PATH$GO_FILENAME"
  	DLCHECK=$?
		# If the download failed, install the package from the distribution
  	if [ "$DLCHECK" != "0" ] ; then
	  	echo ""
	  	echo -e "${YELLOW}[!] COULDN'T DOWNLOAD GO (for $PLATFORM)!${NOCOLOR}"
	  	echo -e "${RED}[+] The go repositories may be blocked or offline!${NOCOLOR}"
	  	echo -e "${RED}[+] We try to install the distribution package, instead.${NOCOLOR}"
	  	echo
		fi
	else
		echo ""
		echo -e "${YELLOW}[!] COULDN'T DOWNLOAD GO because the platform is unknown!${NOCOLOR}"
		echo -e "${RED}[+] We try to install the distribution package, instead.${NOCOLOR}"
		echo
	fi
	if [ "$PLATFORM" == "" ] || [ "$DLCHECK" != "0" ]; then
  	if [ "$STEP_BY_STEP" = "--step_by_step" ]; then
	  	echo ""
	  	read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
	  	clear
  	else
	  	sleep 10
  	fi
	  re-connect
	  sudo apt-get -y install golang
	  GO_PROGRAM="/usr/local/go/bin/go"
	  if [ -f $GO_PROGRAM ]; then
		  GO_VERSION_NR=$($GO_PROGRAM version | cut -d ' ' -f3 | cut -d '.' -f2)
	  else
		  GO_PROGRAM=go
		  #This can lead to command not found - ignore it
		  GO_VERSION_NR=$($GO_PROGRAM version | cut -d ' ' -f3 | cut -d '.' -f2)
	  fi
	  if [ "$GO_VERSION_NR" -lt "17" ]; then
		  echo ""
		  echo -e "${YELLOW}[!] TOO LOW GO VERSION NUMBER${NOCOLOR}"
		  echo -e "${RED}[+] At least go version 1.17 is needed to compile pluggable ${NOCOLOR}"
		  echo -e "${RED}[+] transports. We tried several ways to get a newer go version, ${NOCOLOR}"
		  echo -e "${RED}[+] but failed. Please, try it again later or install go manually. ${NOCOLOR}"
		  echo ""
		  exit 1
	  fi
  else
    sudo tar -C /usr/local -xzvf $GO_FILENAME
	  sudo rm $GO_FILENAME
  fi

	# What if .profile doesn't exist?
  if [ -f ".profile" ]; then
	  if ! grep "Added by TorBox (001)" .profile ; then
		  sudo printf "\n# Added by TorBox (001)\nexport PATH=$PATH:/usr/local/go/bin\n" | tee -a .profile
	  fi
  else
	  sudo printf "\n# Added by TorBox (001)\nexport PATH=$PATH:/usr/local/go/bin\n" | tee -a .profile
  fi
  export PATH=$PATH:/usr/local/go/bin

  if [ "$STEP_BY_STEP" = "--step_by_step" ]; then
    echo ""
    read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
    clear
  else
    sleep 10
  fi
fi

if [ "$STEP_NUMBER" -le "5" ]; then
  # 5. Installing tor
  clear
  echo -e "${RED}[+] Step 5: Installing Tor...${NOCOLOR}"
	re-connect
  select_and_install_tor
	check_install_packages "libtorsocks:armhf"

  if [ "$STEP_BY_STEP" = "--step_by_step" ]; then
	  echo ""
	  read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
	  clear
  else
	  sleep 10
  fi
fi

if [ "$STEP_NUMBER" -le "6" ]; then
  # 6. Configuring Tor with its pluggable transports
  clear
  echo -e "${RED}[+] Step 6: Configuring Tor with its pluggable transports....${NOCOLOR}"
  cd
  git clone $OBFS4PROXY_USED
  DLCHECK=$?
  if [ $DLCHECK -eq 0 ]; then
		if [ "$TORBOX_MINI" == "--torbox_mini" ]; then
			export GOARCH=arm
			export GOARM=6
		fi
	  export GO111MODULE="on"
		cd obfs4proxy
	  go build -o obfs4proxy/obfs4proxy ./obfs4proxy
	  sudo cp ./obfs4proxy/obfs4proxy /usr/bin
	  cd
	  sudo rm -rf obfs4proxy
	  sudo rm -rf go*
  else
	  echo ""
	  echo -e "${YELLOW}[!] COULDN'T CLONE THE OBFS4PROXY REPOSITORY!${NOCOLOR}"
	  echo -e "${RED}[+] The obfs4proxy repository may be blocked or offline!${NOCOLOR}"
	  echo -e "${RED}[+] Please try again later and if the problem persists, please report it${NOCOLOR}"
	  echo -e "${RED}[+] to ${YELLOW}anonym@torbox.ch${RED}. ${NOCOLOR}"
	  echo -e "${RED}[+] In the meantime, we install the distribution package, which may be outdated.${NOCOLOR}"
	  echo ""
	  read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
	  check_install_packages obfs4proxy
	  clear
  fi
  sudo setcap 'cap_net_bind_service=+ep' /usr/bin/obfs4proxy
  (sudo mv /usr/local/bin/tor* /usr/bin) 2>/dev/null
  sudo chmod a+x /usr/share/tor/geoip*
  # Copying not moving!
  (sudo cp /usr/share/tor/geoip* /usr/bin) 2>/dev/null
  sudo sed -i "s/^NoNewPrivileges=yes/NoNewPrivileges=no/g" /lib/systemd/system/tor@default.service
  sudo sed -i "s/^NoNewPrivileges=yes/NoNewPrivileges=no/g" /lib/systemd/system/tor@.service

  if [ "$STEP_BY_STEP" = "--step_by_step" ]; then
	  echo ""
	  read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
	  clear
  else
	  sleep 10
  fi
fi

if [ "$STEP_NUMBER" -le "7" ]; then
  # 7. Install Snowflake
  clear
  echo -e "${RED}[+] Step 7: Installing Snowflake...${NOCOLOR}"
  echo -e "${RED}[+]         This can take some time, please be patient!${NOCOLOR}"
  cd
  git clone $SNOWFLAKE_ORIGINAL_WEB
  DLCHECK=$?
  if [ $DLCHECK -eq 0 ]; then
		if [ "$TORBOX_MINI" == "--torbox_mini" ]; then
			export GOARCH=arm
			export GOARM=6
		fi
	  export GO111MODULE="on"
	  cd snowflake/proxy
	  go get
	  go build
	  sudo cp proxy /usr/bin/snowflake-proxy
	  cd
	  cd snowflake/client
	  go get
	  go build
	  sudo cp client /usr/bin/snowflake-client
	  cd
	  sudo rm -rf snowflake
	  sudo rm -rf go*
  else
	  echo ""
	  echo -e "${YELLOW}[!] COULDN'T CLONE THE SNOWFLAKE REPOSITORY!${NOCOLOR}"
	  echo -e "${RED}[+] The Snowflake repository may be blocked or offline!${NOCOLOR}"
	  echo -e "${RED}[+] Please try again later and if the problem persists, please report it${NOCOLOR}"
	  echo -e "${RED}[+] to ${YELLOW}anonym@torbox.ch${RED}. ${NOCOLOR}"
	  echo ""
	  read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
	  clear
  fi
  if [ "$STEP_BY_STEP" = "--step_by_step" ]; then
	  echo ""
	  read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
	  clear
  else
	  sleep 10
  fi
fi

if [ "$STEP_NUMBER" -le "8" ]; then
  # 8. Again checking connectivity
  clear
  echo -e "${RED}[+] Step 8: Re-checking Internet connectivity${NOCOLOR}"
  re-connect
fi

if [ "$STEP_NUMBER" -le "9" ]; then
  # 9. Downloading and installing TorBox
  sleep 10
  clear
  echo -e "${RED}[+] Step 9: Downloading and installing the latest version of TorBox...${NOCOLOR}"
  echo -e "${RED}[+]          Selected branch ${YELLOW}$TORBOXMENU_BRANCHNAME${RED}...${NOCOLOR}"
  cd
  wget $TORBOXURL
  DLCHECK=$?
  if [ $DLCHECK -eq 0 ] ; then
	  echo -e "${RED}[+]         TorBox' menu Successfully downloaded... ${NOCOLOR}"
	  echo -e "${RED}[+]         Unpacking TorBox menu...${NOCOLOR}"
	  unzip $TORBOXMENU_BRANCHNAME.zip
	  echo ""
	  echo -e "${RED}[+]         Removing the old one...${NOCOLOR}"
	  (rm -r torbox) 2>/dev/null
	  echo -e "${RED}[+]         Moving the new one...${NOCOLOR}"
	  mv TorBox-$TORBOXMENU_BRANCHNAME torbox
	  echo -e "${RED}[+]         Cleaning up...${NOCOLOR}"
	  (rm -r $TORBOXMENU_BRANCHNAME.zip) 2>/dev/null
	  echo ""
  else
	  echo ""
	  echo -e "${YELLOW}[!] COULDN'T DOWNLOAD TORBOX!${NOCOLOR}"
	  echo -e "${RED}[+] The TorBox repositories may be blocked or offline!${NOCOLOR}"
	  echo -e "${RED}[+] Please try again later and if the problem persists, please report it${NOCOLOR}"
	  echo -e "${RED}[+] to ${YELLOW}anonym@torbox.ch${RED}. ${NOCOLOR}"
	  echo ""
	  read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
	  exit 0
  fi

  if [ "$STEP_BY_STEP" = "--step_by_step" ]; then
	  echo ""
	  read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
	  clear
  else
	  sleep 10
  fi
fi

if [ "$STEP_NUMBER" -le "10" ]; then
  # 10. Installing all configuration files
  clear
  cd torbox
  echo -e "${RED}[+] Step 10: Installing all configuration files....${NOCOLOR}"
  echo ""
  (sudo cp /etc/default/hostapd /etc/default/hostapd.bak) 2>/dev/null
  sudo cp etc/default/hostapd /etc/default/
  echo -e "${RED}[+]${NOCOLOR}         Copied /etc/default/hostapd -- backup done"
  (sudo cp /etc/default/isc-dhcp-server /etc/default/isc-dhcp-server.bak) 2>/dev/null
  sudo cp etc/default/isc-dhcp-server /etc/default/
  echo -e "${RED}[+]${NOCOLOR}         Copied /etc/default/isc-dhcp-server -- backup done"
  (sudo cp /etc/dhcp/dhclient.conf /etc/dhcp/dhclient.conf.bak) 2>/dev/null
  sudo cp etc/dhcp/dhclient.conf /etc/dhcp/
  echo -e "${RED}[+]${NOCOLOR}         Copied /etc/dhcp/dhclient.conf -- backup done"
  (sudo cp /etc/dhcp/dhcpd.conf /etc/dhcp/dhcpd.conf.bak) 2>/dev/null
  if [ "$TORBOX_MINI" == "--torbox_mini" ]; then
    sudo cp etc/dhcp/dhcpd-mini.conf /etc/dhcp/dhcpd.conf
  else
    sudo cp etc/dhcp/dhcpd.conf /etc/dhcp/
  fi
  echo -e "${RED}[+]${NOCOLOR}         Copied /etc/dhcp/dhcpd.conf -- backup done"
  (sudo cp /etc/hostapd/hostapd.conf /etc/hostapd/hostapd.conf.bak) 2>/dev/null
  sudo cp etc/hostapd/hostapd.conf /etc/hostapd/
  echo -e "${RED}[+]${NOCOLOR}         Copied /etc/hostapd/hostapd.conf -- backup done"
  (sudo cp /etc/iptables.ipv4.nat /etc/iptables.ipv4.nat.bak) 2>/dev/null
	if [ "$ON_A_CLOUD" == "--on_a_cloud" ]; then sudo cp etc/iptables.ipv4-cloud.nat /etc/iptables.ipv4.nat
	elif [ "$TORBOX_MINI" == "--torbox_mini" ]; then sudo cp etc/iptables.ipv4-mini.nat /etc/iptables.ipv4.nat
	else sudo cp etc/iptables.ipv4.nat /etc/; fi
  echo -e "${RED}[+]${NOCOLOR}         Copied /etc/iptables.ipv4.nat -- backup done"
  (sudo cp /etc/motd /etc/motd.bak) 2>/dev/null
  sudo cp etc/motd /etc/
  echo -e "${RED}[+]${NOCOLOR}         Copied /etc/motd -- backup done"

	# NEW v.0.5.5: TorBox on a Cloud
	# We do not change our /etc/network/interfaces
	(sudo cp /etc/network/interfaces /etc/network/interfaces.bak) 2>/dev/null
  if [ "$ON_A_CLOUD" == "--on_a_cloud" ]; then
		sudo cp etc/network/interfaces /etc/network/
	elif [ "$TORBOX_MINI" == "--torbox_mini" ]; then
		sudo cp etc/network/interfaces.mini /etc/network/interfaces
	else
		sudo cp etc/network/interfaces /etc/network/
	fi
	echo -e "${RED}[+]${NOCOLOR}         Copied /etc/network/interfaces -- backup done"

  sudo cp etc/systemd/system/rc-local.service /etc/systemd/system/rc-local.service
  (sudo cp /etc/rc.local /etc/rc.local.bak) 2>/dev/null
  if [ "$TORBOX_MINI" == "--torbox_mini" ]; then
    sudo cp etc/rc.local.mini /etc/rc.local
  else
    sudo cp etc/rc.local /etc/
  fi
  sudo chmod a+x /etc/rc.local
  echo -e "${RED}[+]${NOCOLOR}         Copied /etc/rc.local -- backup done"
	# NEW v.0.5.5: Configuring IPv4 forwarding in sysctl.d, which replaces /etc/sysctl.conf on Debian Trixie
	echo 'net.ipv4.ip_forward=1' | sudo tee /etc/sysctl.d/99-ipforward.conf
	sudo sysctl -p /etc/sysctl.d/99-ipforward.conf
	echo -e "${RED}[+]${NOCOLOR}         IPv4 forwarding configured!"
  # NEW v.0.5.4: Cloudspecific torrc
  (sudo cp /etc/tor/torrc /etc/tor/torrc.bak) 2>/dev/null
  if [ "$ON_A_CLOUD" == "--on_a_cloud" ]; then sudo cp etc/tor/torrc-cloud /etc/tor/torrc
  else sudo cp etc/tor/torrc /etc/tor/ ; fi
  echo -e "${RED}[+]${NOCOLOR}         Copied /etc/tor/torrc -- backup done"
  echo -e "${RED}[+]${NOCOLOR}         Activating IP forwarding"
  sudo sh -c "echo 1 > /proc/sys/net/ipv4/ip_forward"
  (sudo cp /etc/nginx/nginx.conf /etc/nginx/nginx.conf.bak) 2>/dev/null
  (sudo cp etc/nginx/nginx.conf /etc/nginx/) 2>/dev/null
  echo -e "${RED}[+]${NOCOLOR}         Copied /etc/nginx/nginx.conf -- backup done"
  echo ""

  #Back to the home directory
  cd
	# What if .profile doesn't exist?
  if [ -f ".profile" ]; then
	  if ! grep "Added by TorBox (002)" .profile ; then
		  sudo printf "\n# Added by TorBox (002)\ncd torbox\n./menu\n" | tee -a .profile
	  fi
  else
	  printf "\n# Added by TorBox (002)\ncd torbox\n./menu\n" | tee -a .profile
  fi

  echo -e "${RED}[+]          Make tor ready for Onion Services${NOCOLOR}"
  (sudo mkdir /var/lib/tor/services) 2>/dev/null
  sudo chown -R debian-tor:debian-tor /var/lib/tor/services
  sudo chmod -R go-rwx /var/lib/tor/services
  (sudo mkdir /var/lib/tor/onion_auth) 2>/dev/null
  sudo chown -R debian-tor:debian-tor /var/lib/tor/onion_auth
  sudo chmod -R go-rwx /var/lib/tor/onion_auth

  if [ "$STEP_BY_STEP" = "--step_by_step" ]; then
	  echo ""
	  read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
	  clear
  else
	  sleep 10
  fi
fi

if [ "$STEP_NUMBER" -le "11" ]; then
  # 11. Disabling Bluetooth
  echo -e "${RED}[+] Step 11: Because of security considerations, we completely disable Bluetooth functionality, if available${NOCOLOR}"
  if [ -f "${CONFIGFILE}" ] ; then
    if ! grep "# Added by TorBox" ${CONFIGFILE} ; then
      sudo printf "\n# Added by TorBox\ndtoverlay=disable-bt\n" | sudo tee -a ${CONFIGFILE}
    fi
  fi
	# NEW v.0.5.5: hciuart removed because it is not anymore used in the latest Raspberry Pi OS
	sudo systemctl stop bluetooth.service
	sudo systemctl disable bluetooth.service
	sudo systemctl mask bluetooth.service
	sudo systemctl daemon-reload
  sudo apt-get purge -y bluez bluez-firmware pi-bluetooth
  sudo apt-get -y autoremove
  sudo rfkill block bluetooth
  if [ "$STEP_BY_STEP" = "--step_by_step" ]; then
	  echo ""
	  read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
	  clear
  else
	  sleep 10
  fi
fi

if [ "$STEP_NUMBER" -le "12" ]; then
  # 12. Configure the system services
  clear
  echo -e "${RED}[+] Step 12: Configure the system services...${NOCOLOR}"
  sudo systemctl daemon-reload
  if [ "$TORBOX_MINI" == "--torbox_mini" ] || [ "$ON_A_CLOUD" == "--on_a_cloud" ]; then
    sudo systemctl stop hostapd
    sudo systemctl disable hostapd
    sudo systemctl mask hostapd
  else
    sudo systemctl unmask hostapd
    sudo systemctl enable hostapd
    sudo systemctl start hostapd
  fi
  sudo systemctl unmask isc-dhcp-server
  sudo systemctl enable isc-dhcp-server
  sudo systemctl start isc-dhcp-server
  sudo systemctl stop tor
  sudo systemctl mask tor
  # Both tor services have to be masked to block outgoing tor connections
  sudo systemctl mask tor@default.service
  sudo systemctl unmask ssh
  sudo systemctl enable ssh
  sudo systemctl start ssh
  echo ""
  echo -e "${RED}[+]          Stop logging, now...${NOCOLOR}"
  sudo systemctl stop rsyslog
  sudo systemctl disable rsyslog
  sudo systemctl mask rsyslog
  sudo systemctl stop systemd-journald-dev-log.socket
  sudo systemctl stop systemd-journald-audit.socket
  sudo systemctl stop systemd-journald.socket
  sudo systemctl stop systemd-journald.service
  sudo systemctl mask systemd-journald.service
  echo""

  # Make Nginx ready for Webssh and Onion Services
  echo -e "${RED}[+]          Make Nginx ready for Webssh and Onion Services${NOCOLOR}"
  sudo systemctl stop nginx
  (sudo rm /etc/nginx/sites-enabled/default) 2>/dev/null
  (sudo rm /etc/nginx/sites-available/default) 2>/dev/null
  (sudo rm -r /var/www/html) 2>/dev/null
  # This is necessary for Nginx / TFS
  (sudo chown torbox:torbox /var/www) 2>/dev/null
  # Configuring webssh
  sudo cp torbox/etc/nginx/sites-available/sample-webssh.conf /etc/nginx/sites-available/webssh.conf
  sudo ln -sf /etc/nginx/sites-available/webssh.conf /etc/nginx/sites-enabled/
  # HAS TO BE TESTED: https://unix.stackexchange.com/questions/164866/nginx-leaves-old-socket
  (sudo sed "s|STOP_SCHEDULE=\"${STOP_SCHEDULE:-QUIT/5/TERM/5/KILL/5}\"|STOP_SCHEDULE=\"${STOP_SCHEDULE:-TERM/5/KILL/5}\"|g" /etc/init.d/nginx) 2>/dev/null
  #sudo systemctl start nginx
  sudo systemctl daemon-reload

  if [ "$STEP_BY_STEP" = "--step_by_step" ]; then
	  echo ""
	  read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
	  clear
  else
	  sleep 10
  fi
fi

if [ "$STEP_NUMBER" -le "13" ]; then
  # 13. Updating run/torbox.run
  clear
  echo -e "${RED}[+] Step 13: Configuring TorBox and update run/torbox.run...${NOCOLOR}"
  echo -e "${RED}[+]          Update run/torbox.run${NOCOLOR}"
  sudo sed -i "s/^NAMESERVERS=.*/NAMESERVERS=${NAMESERVERS_ORIG}/g" ${RUNFILE}
  sudo sed -i "s|^GO_DL_PATH=.*|GO_DL_PATH=${GO_DL_PATH}|g" ${RUNFILE}
  sudo sed -i "s|^OBFS4PROXY_USED=.*|OBFS4PROXY_USED=${OBFS4PROXY_USED}|g" ${RUNFILE}
  sudo sed -i "s|^SNOWFLAKE_USED=.*|SNOWFLAKE_USED=${SNOWFLAKE_ORIGINAL_WEB}|g" ${RUNFILE}
  # NEW v.0.5.4: Specifc configurations for an installation on a cloud and for TorBox mini
  # Important: Randomizing MAC addresses could prevent the assignement of an IP address on a cloud
  if [ "$ON_A_CLOUD" == "--on_a_cloud" ]; then
	  sudo sed -i "s/^FRESH_INSTALLED=.*/FRESH_INSTALLED=1/" ${RUNFILE}
		sudo sed -i "s/^SSH_FROM_INTERNET=.*/SSH_FROM_INTERNET=1/" ${RUNFILE}
	  sudo sed -i "s/^ON_A_CLOUD=.*/ON_A_CLOUD=1/" ${RUNFILE}
    sudo sed -i "s/^TORBOX_MINI=.*/TORBOX_MINI=0/" ${RUNFILE}
		sudo sed -i "s/^TORBOX_MINI_DEFAULT=.*/TORBOX_MINI_DEFAULT=0/" ${RUNFILE}
	  sudo sed -i "s/=random/=permanent/" ${RUNFILE}
  elif [ "$TORBOX_MINI" == "--torbox_mini" ]; then
    sudo sed -i "s/^FRESH_INSTALLED=.*/FRESH_INSTALLED=3/" ${RUNFILE}
		sudo sed -i "s/^SSH_FROM_INTERNET=.*/SSH_FROM_INTERNET=0/" ${RUNFILE}
    sudo sed -i "s/^ON_A_CLOUD=.*/ON_A_CLOUD=0/" ${RUNFILE}
		sudo sed -i "s/^TORBOX_MINI_DEFAULT=.*/TORBOX_MINI_DEFAULT=1/" ${RUNFILE}
    sudo sed -i "s/^TORBOX_MINI=.*/TORBOX_MINI=1/" ${RUNFILE}
  else
	  sudo sed -i "s/^FRESH_INSTALLED=.*/FRESH_INSTALLED=3/" ${RUNFILE}
		sudo sed -i "s/^SSH_FROM_INTERNET=.*/SSH_FROM_INTERNET=0/" ${RUNFILE}
	  sudo sed -i "s/^ON_A_CLOUD=.*/ON_A_CLOUD=0/" ${RUNFILE}
		sudo sed -i "s/^TORBOX_MINI_DEFAULT=.*/TORBOX_MINI_DEFAULT=0/" ${RUNFILE}
    sudo sed -i "s/^TORBOX_MINI=.*/TORBOX_MINI=0/" ${RUNFILE}
  fi

  echo -e "${RED}[+]          Update sudo setup${NOCOLOR}"
  sudo mkdir /home/torbox/openvpn
  sudo chown -R torbox:torbox /home/torbox/
  if ! sudo grep "# Added by TorBox" /etc/sudoers ; then
    sudo printf "\n# Added by TorBox\ntorbox  ALL=(ALL) NOPASSWD: ALL\n" | sudo tee -a /etc/sudoers
    (sudo visudo -c) 2>/dev/null
  fi
  cd /home/torbox/

  if [ "$STEP_BY_STEP" = "--step_by_step" ]; then
	  echo ""
	  read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
	  clear
  else
	  sleep 10
  fi
fi

if [ "$STEP_NUMBER" -le "14" ]; then
  #14.  TorBox mini specific configurations
  if [ "$TORBOX_MINI" == "--torbox_mini" ]; then
    if ! grep "dwc2,g_ether" ${CMDLINEFILE}; then
      if grep "modules-load" ${CMDLINEFILE}; then
        CMDLINE_STRING=$(grep -o "modules-load=.*" ${CMDLINEFILE} | cut -d ' ' -f 1)
        CMDLINE_STRING_NEW="$CMDLINE_STRING,dwc2,g_ether"
        sudo sed -i "s|${CMDLINE_STRING}|${CMDLINE_STRING_NEW}|g" ${CMDLINEFILE}
      else
        sudo sed -i "s|rootwait|modules-load=dwc2,g_ether rootwait|g" ${CMDLINEFILE}
      fi
    fi
  	if ! grep "dwc2,dr_mode=peripheral" ${CONFIGFILE}; then
			(printf "\ndtoverlay=dwc2,dr_mode=peripheral\n" | sudo tee -a ${CONFIGFILE}) >/dev/null 2>&1
  	fi
		clear
		echo -e "${RED}[+] Step 14: TorBox is configured to be used in a Raspberry Pi Zero 2 W${NOCOLOR}"
		if [ "$STEP_BY_STEP" = "--step_by_step" ]; then
			echo ""
			read -n 1 -s -r -p $'\e[1;31mPlease press any key to continue... \e[0m'
			clear
		else
			sleep 10
		fi
	fi
fi

if [ "$STEP_NUMBER" -le "15" ]; then
  # 15. Finishing, cleaning and booting
	clear
	echo -e "${RED}[+] Step 15: We are finishing and cleaning up now!${NOCOLOR}"
  echo -e "${RED}[+]          This will erase all log files and cleaning up the system.${NOCOLOR}"
  echo ""
  echo -e "${YELLOW}[!] IMPORTANT${NOCOLOR}"
  echo -e "${YELLOW}    After this last step, TorBox has to be rebooted.${NOCOLOR}"
  echo -e "${YELLOW}    Afterwards, log in with \"torbox\" and your choosen password !! ${NOCOLOR}"
  echo -e "${YELLOW}    If connecting via TorBox's WiFi (TorBox055) use \"CHANGE-IT\" as password.${NOCOLOR}"
  echo ""
  read -n 1 -s -r -p $'\e[1;31mTo complete the installation, please press any key... \e[0m'
  clear
  echo -e "${RED}[+] Erasing big not usefull packages...${NOCOLOR}"
  # Find the bigest space waster packages: dpigs -H
  sudo apt-get -y --purge remove exim4 exim4-base exim4-config exim4-daemon-light
  sudo apt-get -y remove libgl1-mesa-dri texlive* lmodern
  sudo apt-get -y clean
  sudo apt-get -y autoclean
  sudo apt-get -y autoremove
  echo -e "${RED}[+] Setting the timezone to UTC${NOCOLOR}"
  sudo timedatectl set-timezone UTC
  echo -e "${RED}[+] Erasing ALL LOG-files...${NOCOLOR}"
  echo " "
  # shellcheck disable=SC2044
  for logs in $(sudo find /var/log -type f); do
    echo -e "${RED}[+]${NOCOLOR} Erasing $logs"
    sudo rm $logs
    sleep 1
  done
	sudo journalctl --vacuum-size=1M
  echo -e "${RED}[+]${NOCOLOR} Erasing History..."
	(rm .bash_history) 2>/dev/null
  history -c
  # To start TACA, notices.log has to be present
  (sudo -u debian-tor touch /var/log/tor/notices.log) 2>/dev/null
  (sudo chmod -R go-rwx /var/log/tor/notices.log) 2>/dev/null
  echo ""
  echo -e "${RED}[+]${NOCOLOR} Disable auto-login..."
  sudo raspi-config nonint do_boot_behaviour B1
  echo ""
  echo -e "${RED}[+] Setting up the hostname...${NOCOLOR}"
  # This has to be at the end to avoid unnecessary error messages
  (sudo hostnamectl set-hostname "$HOSTNAME") 2>/dev/null
  (sudo systemctl restart systemd-hostnamed) 2>/dev/null
  if grep 127.0.1.1.* /etc/hosts ; then
	  (sudo sed -i "s/127.0.1.1.*/127.0.1.1\t$HOSTNAME/g" /etc/hosts) 2>/dev/null
  else
	  (sudo sed -i "s/^::1/127.0.1.1\t$HOSTNAME\n::1/g" /etc/hosts) 2>/dev/null
  fi
  #
  # OLD
  echo ""
  echo -e "${YELLOW}[!] IMPORTANT${NOCOLOR}"
  echo -e "${YELLOW}    TorBox has to be rebooted.${NOCOLOR}"
  echo -e "${YELLOW}    Afterwards, log in with \"torbox\" and your choosen password !! ${NOCOLOR}"
  echo -e "${YELLOW}    If connecting via TorBox's WiFi (TorBox055) use \"CHANGE-IT\" as password.${NOCOLOR}"
  echo ""

  if [ "$STEP_BY_STEP" = "--step_by_step" ]; then
	  echo ""
	  read -n 1 -s -r -p $'\e[1;31mPlease press any key to REBOOT... \e[0m'
	  clear
  else
	  sleep 10
  fi
fi

echo -e "${RED}[+] Rebooting...${NOCOLOR}"
sync
sleep 3
sudo reboot
