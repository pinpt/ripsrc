#!/bin/bash
CommandResult="Interface    Chipset     Driver     mon0    Unknown      iwlwifi - [phy0]wlan0       Unknown     iwlwifi - [phy0]"
InstanceId="mon0";
count=`grep -o "$InstanceId" <<< "$CommandResult" | wc -l`
echo "$InstanceId encountered "$count" times";
