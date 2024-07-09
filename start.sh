#! /bin/bash

docker start ollama
docker run -d --network host eclipse-mosquitto
