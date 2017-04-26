# Metrics Flow Daemon ![Build Status](https://travis-ci.org/QubitProducts/mflowd.svg) ![Go Report](https://goreportcard.com/badge/github.com/QubitProducts/mflowd)

It is a small daemon that aggregates custom monitoring metrics collected from Google Dataflow workers that use [metrics-flow](https://github.com/QubitProducts/metrics-flow) library and exposes them to [Prometheus](https://prometheus.io/).

# How it works

Google Dataflow workers with `metrics-flow` plugged in pre-aggregate custom monitoring metrics using pipeline windowing functions and dump
the results into a Google Pub/Sub topic. The metrics flow daemon polls a subscription to the topic, converts received metric update events to `Prometheus` format and exposes them through `/metrics` endpoint.


    +------------+                                                      
    | Dataflow 1 +----+                                                 
    +------------+    |                                                 
                      |                                                 
    +------------+    |                       +------------+            
    | Dataflow i |----|->(Google Pub/Sub)---->|   mflowd   |            
    +------------+    |                       +------------+            
                      |                              ^                  
    +------------+    |                              |                  
    | Dataflow N |----+                       +-------------+           
    +------------+                            | Prometheus  |           
                                              +-------------+           
                                                                   
                                                                    
                                                                                                                  
# Installation

    % go get github.com/QubitProducts/mflowd

## Build it from scratch

1. Make sure you have `glide` installed (if you don't know how to install it, follow [this](https://github.com/Masterminds/glide#install) link)
2. `make bootstrap`
3. `make test`
4. `make mflowd`

# Running

1. Create a [pub/sub topic](https://cloud.google.com/pubsub/docs/publisher#create) you will use for publishing metrics from your Dataflow workers (if you don't have one already).
2. Create a [pull subscription](https://cloud.google.com/pubsub/docs/pull) to the topic
3. Make sure you are authorized to use the subscription (if not sure, use [gcloud auth login](https://cloud.google.com/sdk/gcloud/reference/auth/login))
4. Run the daemon

       % ./mflowd [-v] -p <port> -s pubsub <subscription_id>

Where
* `port` is a port where `/metrics` endpoint will be exposed
* `subscription_path` is a subscription identifier which usually looks like `projects/<project_name>/subscriptions/<subcsciption_name>`
* use optional `-v` flag to run the daemon in verbose mode

# Using Docker 

You can easily build a "containerized" version of `mflowd` and run it on `mesos` or `kubernetes`. 

## Building

    % make docker
    % docker images | grep mflowd
    mflowd                                             latest                                            e9cbac93f703
    ...

## Running

Before you can run the image you need to set up a Google Cloud API service account to allow `mflowd` use the subscription you have created. So

1. [Create a service account](https://cloud.google.com/vision/docs/common/auth) for mflowd
2. Create an empty directory on your host machine (say, `% mkdir ~/.mflowd`)
3. Download the service account key in JSON format and put it to the created directory
4. Finally run the countainer:

       % docker run -e "MFLOWD_SUB=<subscription_id>" -v $HOME/.mflowd:/etc/mflowd 'mflowd:latest'

## Using docker-compose

You can also run both `mflowd` and `prometheus` docker images using `docker-compose`:

    % cd ~/go/src/github.com/QubitProducts/mflowd
    % mkdir gcp
    # download your service account JSON key to gcp directory
    % cat > .env
    MFLOWD_SUB=<subscription_id>
    MFLOWD_VERBOSE=0 # set to 1 to turn verbose mode on
    ^C
    % docker-compose up

Follow http://localhost:9090 to get to `Prometheus` UI
