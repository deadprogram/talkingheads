#! /bin/bash

docker start ollama
#docker exec ollama ollama run gemma2
#docker exec ollama ollama run dagbs/dolphin-2.9.2-phi-3-medium:iq3_xxs
#docker exec ollama ollama run Lexi-Llama-3-8B-Uncensored_Q4_K_M
docker run -d --network host eclipse-mosquitto
