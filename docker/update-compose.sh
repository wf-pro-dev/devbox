ARG_SERVICE=$1
ARG_TAG=$2

SERVICE=${ARG_SERVICE:-:-api}
TAG=${ARG_TAG:-${DEVBOX_BACKEND_VERSION:-latest}}

PROJECT_DIR="$HOME/devbox"
DOCKER_DIR="$PROJECT_DIR/docker"
BUILD="Dockerfile.$SERVICE"
IMAGE="devbox-$SERVICE:latest"
TAG_IMAGE="wwwill-1.lab:5000/devbox/$SERVICE:$TAG"

# build the new image
docker build -f "$DOCKER_DIR/$BUILD" -t $IMAGE ../

# Save the new IMAGE
docker tag $IMAGE $TAG_IMAGE

# Push the new image to repository
docker push $TAG_IMAGE

# Start new container
docker compose up -d $SERVICE