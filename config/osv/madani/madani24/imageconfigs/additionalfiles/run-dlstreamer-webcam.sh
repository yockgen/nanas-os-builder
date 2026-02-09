#!/bin/bash
set -e

export MODELS_PATH=/data/intel/models
export XSOCK=/tmp/.X11-unix
export XAUTH=${XAUTHORITY:-$HOME/.Xauthority}

# Crucial: Allow local connections to X11
xhost +local:docker > /dev/null

docker run -it --rm \
    --net=host \
    --name dlstreamer-demo \
    --privileged \
    --group-add keep-groups \
    --device /dev/dri:/dev/dri \
    --device /dev/video0:/dev/video0:rw \
    -e DISPLAY=$DISPLAY \
    -e XAUTHORITY=$XAUTH \
    -v $XSOCK:$XSOCK \
    -v $XAUTH:$XAUTH \
    -v ${MODELS_PATH}:/home/dlstreamer/models:Z \
    intel/dlstreamer:latest \
    /bin/bash -c "gst-launch-1.0 \
        v4l2src device=/dev/video0 ! \
        video/x-raw,format=YUY2,width=640,height=480 ! \
        videoconvert ! \
        video/x-raw,format=NV12 ! \
        gvadetect model=/home/dlstreamer/models/intel/face-detection-adas-0001/FP32/face-detection-adas-0001.xml \
        device=CPU threshold=0.85 ! \
        gvaclassify model=/home/dlstreamer/models/intel/emotions-recognition-retail-0003/FP32/emotions-recognition-retail-0003.xml \
        device=CPU ! \
        gvawatermark ! videoconvert ! ximagesink sync=false"