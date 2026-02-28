#!/bin/bash
set -e

export MODELS_PATH=/data/intel/models
export XSOCK=/tmp/.X11-unix
export XAUTH=${XAUTHORITY:-$HOME/.Xauthority}

# Crucial: Allow local connections to X11
xhost +local:docker > /dev/null

# Sample Intel Carpark video
VIDEO_URL="https://github.com/intel-iot-devkit/sample-videos/raw/master/car-detection.mp4"

docker run -it --rm \
    --net=host \
    --name dlstreamer-carpark \
    --privileged \
    --group-add keep-groups \
    --device /dev/dri:/dev/dri \
    -e DISPLAY=$DISPLAY \
    -e XAUTHORITY=$XAUTH \
    -v $XSOCK:$XSOCK \
    -v $XAUTH:$XAUTH \
    -v ${MODELS_PATH}:/home/dlstreamer/models:Z \
    intel/dlstreamer:latest \
    /bin/bash -c "gst-launch-1.0 \
        urisourcebin uri=$VIDEO_URL ! decodebin ! \
        videoconvert ! \
        video/x-raw,format=NV12 ! \
        gvadetect model=/home/dlstreamer/models/intel/person-vehicle-bike-detection-2004/FP32/person-vehicle-bike-detection-2004.xml \
        device=CPU threshold=0.5 ! \
        gvaclassify model=/home/dlstreamer/models/intel/vehicle-attributes-recognition-barrier-0039/FP32/vehicle-attributes-recognition-barrier-0039.xml \
        device=CPU object-class=vehicle ! \
        gvawatermark ! videoconvert ! ximagesink sync=false"
