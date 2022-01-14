#!/bin/bash

version=$1

tag="korylprince/ping-dashboard:$version"

docker build --no-cache --build-arg "VERSION=$version" --tag "$tag" .

docker push "$tag"

if [ "$2" = "latest" ]; then
    docker tag "$tag" "korylprince/ping-dashboard:latest"
    docker push "korylprince/ping-dashboard:latest"
fi
