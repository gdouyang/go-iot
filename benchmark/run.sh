#!/usr/bin/env bash
java -jar device-simulator.jar \
mqtt.address=192.168.31.197 \
mqtt.port=9010 \
mqtt.limit=10000 \
mqtt.eventLimit=10000 \
mqtt.maxSendTotal=8000000 \
mqtt.start=1111 
#mqtt.batchSize=1000 mqtt.binds=192.168.31.50,192.168.31.51,192.168.31.52
