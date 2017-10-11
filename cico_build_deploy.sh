#!/bin/bash

set -e

BUILD_IMAGE="jenkins-controller-build"
BUILD_CONTAINER=${BUILD_IMAGE}"-container"

DEPLOY_IMAGE="jenkins-controller-deploy"

PROJ_PATH="/root/go/src/github.com/vpavlin/jenkins-controller"
TARGET_DIR="go-binary"


function tag_push() {
    TARGET_IMAGE=$1
    USERNAME=$2
    PASSWORD=$3
    REGISTRY=$4

    docker tag ${DEPLOY_IMAGE} ${TARGET_IMAGE}
    if [ -n "${USERNAME}" ] && [ -n "${PASSWORD}" ]; then
        docker login -u ${USERNAME} -p ${PASSWORD} ${REGISTRY}
    fi
    docker push ${TARGET_IMAGE}

}

if [ -z $CICO_LOCAL ]; then
    [ -f jenkins-env ] && cat jenkins-env | grep -e PASS -e GIT -e DEVSHIFT > inherit-env
    [ -f inherit-env ] && . inherit-env

    # We need to disable selinux for now, XXX
    /usr/sbin/setenforce 0

    # Get all the deps in
    yum -y install docker make git

    # Get all the deps in
    yum -y install docker make git
    service docker start
fi

docker ps | grep -q ${BUILD_CONTAINER} && docker stop ${BUILD_CONTAINER}
docker ps -a | grep -q ${BUILD_CONTAINER} && docker rm ${BUILD_CONTAINER}
rm -rf ${TARGET_DIR}/

docker build -t ${BUILD_IMAGE} -f Dockerfile.build .

mkdir ${TARGET_DIR}/

docker run -t -d -v ${PWD}:${PROJ_PATH} -v ${PWD}/${TARGET_DIR}:/${TARGET_DIR} --name ${BUILD_CONTAINER} ${BUILD_IMAGE} 

echo "==> Getting dependencies"
docker exec -it ${BUILD_CONTAINER} bash -c "cd /root/go/src/ && go get ./..."
echo "==> Building the build image"
docker exec -it ${BUILD_CONTAINER} bash -c "cd ${PROJ_PATH} && go build "

echo "==> Copying the result"
docker exec -it ${BUILD_CONTAINER} bash -c "cp /root/go/bin/* /${TARGET_DIR} && chown $(id -u):$(id -g) /${TARGET_DIR}/*"

echo "==> Building the deploy image"
docker build -t ${DEPLOY_IMAGE} -f Dockerfile.deploy .

if [ -z ${CICO_LOCAL} ]; then
    TAG=$(echo ${GIT_COMMIT} | cut -c1-${DEVSHIFT_TAG_LEN})

    tag_push "${REGISTRY_URL}:${TAG}" ${DEVSHIFT_USERNAME} ${DEVSHIFT_PASSWORD} ${REGISTRY_URI}
    tag_push "${REGISTRY_URL}:latest" ${DEVSHIFT_USERNAME} ${DEVSHIFT_PASSWORD} ${REGISTRY_URI}
fi
