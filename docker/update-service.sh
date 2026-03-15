#!/usr/bin/bash
set -e

usage() {
	echo "Usage: $0 -s <service> -r <replicas> -t <tag>"
	echo "Example: $0 -s api -r 2 -t latest"
	echo ""
	echo "  -s <service>   Service to update (api, sse, nodejs)"
	echo "  -r <replicas>  Number of replicas to scale to"
	echo "  -t <tag>       Tag to use for the image"
	echo "  -h             Show this help message"
	exit 1
}

while getopts "s:r:t:h" opt; do
	case $opt in
		s) ARG_SERVICE=$OPTARG ;;
		r) ARG_REPLICAS=$OPTARG ;;
		t) ARG_TAG=$OPTARG ;;
		h) usage ;;
		*) echo "Invalid option: -$OPTARG" ;;
	esac
done

SERVICE=${ARG_SERVICE:-${1:-api}}
REPLICAS=${ARG_REPLICAS:-1}
TAG=${ARG_TAG:-${IMAGE_TAG:-dev}}

PROJECT_DIR="$HOME/unipilot"
DOCKER_DIR="$PROJECT_DIR/docker"
DEV="dev-$SERVICE"
DEV_CONTAINER="$DEV-1"
DEV_IMAGE="$DEV:latest"
IMAGE="unipilot-$SERVICE:latest"
TAG_IMAGE="wwwill-1.lab:5000/unipilot/$SERVICE:$TAG"
LATEST_IMAGE="wwwill-1.lab:5000/unipilot/$SERVICE:latest"
BUILD="Dockerfile.$SERVICE"

if [ -z $SERVICE ]; then
	echo "ERROR: Enter at least one argument"
	exit 1
fi

# build the new image
docker build -f "$DOCKER_DIR/$BUILD" -t $IMAGE .

# Save the new IMAGE
docker tag $IMAGE $TAG_IMAGE

# Push the new image to repository
docker push $TAG_IMAGE

# Save the lastest IMAGE
docker tag $IMAGE $LATEST_IMAGE

# Push latest IMAGE
docker push $LATEST_IMAGE

# Update the service
docker service update --image $TAG_IMAGE --force "unipilot_$SERVICE"

# Rescale the service
docker service scale "unipilot_$SERVICE"=$REPLICAS

# Review the service
docker service ps "unipilot_$SERVICE"

# Clean up
docker stop $DEV_CONTAINER && docker rm $DEV_CONTAINER

#docker rmi $DEV_IMAGE

docker rmi $IMAGE
