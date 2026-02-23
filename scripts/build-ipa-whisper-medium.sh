#!/bin/bash

################################################################################
# 
#       NEURLANG/IPA-WHISPER-MEDIUM MODEL
# 
################################################################################
# GIT CLONE
################################################################################

git lfs install
git clone https://github.com/ggml-org/ggml
git clone https://huggingface.co/neurlang/ipa-whisper-medium
git clone https://github.com/openai/whisper.git
git clone https://github.com/ggml-org/whisper.cpp.git

# Make sure Vulkan dependencies are installed:
echo "Installing dependencies...."
sudo apt install -y build-essential cmake libvulkan-dev vulkan-tools vulkan-validationlayers libstdc++-12-dev libgomp1

################################################################################
# GGML
################################################################################

cd ggml
git reset --hard v0.9.7

# install python dependencies in a virtual environment
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt

# build the examples
mkdir build 
cd build
cmake ..
cmake --build . --config Release -j 8

cd ../..

################################################################################
# WHISPER.CPP PREPARE convert-h5-to-ggml.py DEPS
################################################################################

pip install torch --index-url https://download.pytorch.org/whl/cpu
pip install transformers safetensors numpy 

################################################################################
# WHISPER.CPP MAKE NEURLANG/IPA-WHISPER-MEDIUM MODEL
################################################################################

cd whisper.cpp
git reset --hard v1.8.3

python3 ./models/convert-h5-to-ggml.py \
    ../ipa-whisper-medium \
    ../whisper \
    .

mv ggml-model.bin models/ggml-base.en.bin

################################################################################
# WHISPER.CPP BUILD
################################################################################

# build the project
cmake -B build -S . \
    -DGGML_VULKAN=ON \        # Enable Vulkan backend
    -DCMAKE_BUILD_TYPE=Release  # Release build
cmake --build build -j --config Release

################################################################################
# WHISPER.CPP BACKTEST
################################################################################

# transcribe an audio file
./build/bin/whisper-cli -f samples/jfk.wav

cd ..
