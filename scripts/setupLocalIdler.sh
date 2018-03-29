#!/usr/bin/env bash
#
# Used to run Idler locally.
#

set -o errexit
set -o pipefail

LOCAL_PROXY_PORT=${LOCAL_PROXY_PORT:-9101}
LOCAL_TENANT_PORT=${LOCAL_TENANT_PORT:-9102}
LOCAL_TOGGLE_PORT=${LOCAL_TOGGLE_PORT:-9103}
LOCAL_DEFAULT_IDLE_TIME=${LOCAL_DEFAULT_IDLE_TIME:-30}

###############################################################################
# Prints help message
# Globals:
#   None
# Arguments:
#   None
# Returns:
#   None
###############################################################################
printHelp() {
    cat << EOF
Usage: ${0##*/} [start|stop|env|unset]

This script is used to run the Jenkins Idler on localhost.

As a prerequisite your OpenShift access token for dsaas-preview must be exported as DSAAS_PREVIEW_TOKEN.
You can get this token by logging in to https://console.rh-idev.openshift.com/ and select the
"Command Line Tools" option. You do need edit permissions for dsaas-preview in order to port-forward.

You also need to export JC_FIXED_UUIDS to limit the users which get affected by this running instance.
This is to avoid creating side-effects with the running Idler in dsaas-preview.

You can determine a users UUID by running:
> curl -sgSL https://api.prod-preview.openshift.io/api/users?filter[email]=john.doe@example.com | jq .data[0].id

Sample usage (in your shell from the root of fabric8-jenkins-idler):

> export DSAAS_PREVIEW_TOKEN=<dsaas-preview token>
> export JC_FIXED_UUIDS=<list of uuids to enable for idling>
> ./scripts/${0##*/} start
> eval \$(./scripts/${0##*/} env)
> fabric8-jenkins-idler

If you want to test with the docker image, after you have "start" this script, eval its environment variable
and build its image with "make image" you can do :

\$ docker run -it --net=host --env-file <(env|grep JC_) push.registry.devshift.net/fabric8-services/fabric8-jenkins-idler

which would run the idler with the local setup.

To stop:

> ./scripts/${0##*/} stop
EOF
}

###############################################################################
# Wraps oc command with namespace and token parameters
# Globals:
#   DSAAS_PREVIEW_TOKEN - token to run commands against staging cluster
# Arguments:
#   Passes all arguments to oc command
# Returns:
#   None
###############################################################################
loc() {
    oc --config $(dirname $0)/config $@
}

###############################################################################
# Forwards the jenkins-proxy service to localhost.
# Globals:
#   LOCAL_PROXY_PORT - local Idler port
# Arguments:
#   None
# Returns:
#   None
###############################################################################
forwardProxyService() {
    pod=$(loc get pods -l deploymentconfig=jenkins-proxy -o json | jq -r '.items[0].metadata.name')
    if [ "${pod}" == "null" ] ; then
        echo "WARN: Unable to determine Proxy pod name"
        return
    fi

    if lsof -Pi :${LOCAL_PROXY_PORT} -sTCP:LISTEN -t >/dev/null ; then
        echo "INFO: Local Proxy port ${LOCAL_PROXY_PORT} already listening. Skipping oc port-forward" >&2
        return
    fi

    while :
    do
	    loc port-forward ${pod} ${LOCAL_PROXY_PORT}:9091
	    echo "Proxy port forward stopped with exit code $?.  Respawning.." >&2
	    sleep 1
    done
    echo "Proxy port forward stopped." >&2
}

###############################################################################
# Forwards the f8tenant service to localhost
# Globals:
#   LOCAL_TENANT_PORT - local tenant port
# Arguments:
#   None
# Returns:
#   None
###############################################################################
forwardTenantService() {
    pod=$(loc get pods -l deploymentconfig=f8tenant -o json | jq -r '.items[0].metadata.name')
    if [ "${pod}" == "null" ] ; then
        echo "WARN: Unable to determine Tenant pod name"
        return
    fi
    port=$(loc get pods -l deploymentconfig=f8tenant -o json | jq -r '.items[0].spec.containers[0].ports[0].containerPort')

    if lsof -Pi :${LOCAL_TENANT_PORT} -sTCP:LISTEN -t >/dev/null ; then
        echo "INFO: Local Tenant port ${LOCAL_TENANT_PORT} already listening. Skipping oc port-forward" >&2
        return
    fi

    while :
    do
	    loc port-forward ${pod} ${LOCAL_TENANT_PORT}:${port}
	    echo "Tenant port forward stopped with exit code $?.  Respawning.." >&2
	    sleep 1
    done
    echo "Tenant port forward stopped." >&2
}

###############################################################################
# Forwards the toggle service to localhost.
# Globals:
#   LOCAL_TOGGLE_PORT - local toggle service port
# Arguments:
#   None
# Returns:
#   None
###############################################################################
forwardToggleService() {
    pod=$(loc get pods -l deploymentconfig=f8toggles -o json | jq -r '.items[0].metadata.name')
    if [ "${pod}" == "null" ] ; then
        echo "WARN: Unable to determine toggle service pod name"
        return
    fi
    port=$(loc get pods -l deploymentconfig=f8toggles -o json | jq -r '.items[0].spec.containers[0].ports[0].containerPort')

    if lsof -Pi :${LOCAL_TOGGLE_PORT} -sTCP:LISTEN -t >/dev/null ; then
        echo "INFO: Local Toggle port ${LOCAL_TOGGLE_PORT} already listening. Skipping oc port-forward" >&2
        return
    fi

    while :
    do
	    loc port-forward ${pod} ${LOCAL_TOGGLE_PORT}:${port}
	    echo "Toggle port forward stopped with exit code $?.  Respawning.." >&2
	    sleep 1
    done
    echo "Toggle port forward stopped." >&2
}

###############################################################################
# Retrieves the required OpenShift and Auth token from
# Globals:
#   JC_SERVICE_ACCOUNT_ID     - Id for authenticating against Auth service
#   JC_SERVICE_ACCOUNT_SECRET - Secret for authenticating against Auth service
#   JC_AUTH_TOKEN_KEY         - Key to decrypt OpenShift API tokens
# Arguments:
#   None
# Returns:
#   None
###############################################################################
setTokens() {
    pod=$(loc get pods -l deploymentconfig=jenkins-idler -o json | jq -r '.items[0].metadata.name')
    if [ "${pod}" == "null" ] ; then
        echo "WARN: Unable to determine Idler pod name"
        return
    fi

    export JC_SERVICE_ACCOUNT_ID=$(loc exec ${pod} env | grep JC_SERVICE_ACCOUNT_ID= | sed -e 's/JC_SERVICE_ACCOUNT_ID=//')
    export JC_SERVICE_ACCOUNT_SECRET=$(loc exec ${pod} env | grep JC_SERVICE_ACCOUNT_SECRET= | sed -e 's/JC_SERVICE_ACCOUNT_SECRET=//')
    export JC_AUTH_TOKEN_KEY=$(loc exec ${pod} env | grep JC_AUTH_TOKEN_KEY= | sed -e 's/JC_AUTH_TOKEN_KEY=//')
}

###############################################################################
# Ensures login to OpenShift
# Globals:
#   None
# Arguments:
#   None
# Returns:
#   None
###############################################################################
login() {
    [ -z "${DSAAS_PREVIEW_TOKEN}" ] && echo "DSAAS_PREVIEW_TOKEN needs to be exported." && exit 1

    loc login https://api.rh-idev.openshift.com -n dsaas-preview --token=${DSAAS_PREVIEW_TOKEN} >/dev/null
}

###############################################################################
# Starts the port forwarding.
# Globals:
#   None
# Arguments:
#   None
# Returns:
#   None
###############################################################################
start() {
    [ -z "${DSAAS_PREVIEW_TOKEN}" ] && echo "DSAAS_PREVIEW_TOKEN needs to be exported." && exit 1

    login
    forwardProxyService &
    forwardTenantService &
    forwardToggleService &
}

###############################################################################
# Displays the required environment settings for evaluation.
# Globals:
#   None
# Arguments:
#   None
# Returns:
#   None
###############################################################################
env() {
    [ -z "${DSAAS_PREVIEW_TOKEN}" ] && echo "DSAAS_PREVIEW_TOKEN needs to be exported." && exit 1
    [ -z "${JC_FIXED_UUIDS}" ] && echo "JC_FIXED_UUIDS needs to be exported." && exit 1

    login
    setTokens

    echo export JC_IDLE_AFTER=${LOCAL_DEFAULT_IDLE_TIME}
    echo export JC_JENKINS_PROXY_API_URL=http://localhost:${LOCAL_PROXY_PORT}
    echo export JC_F8TENANT_API_URL=http://localhost:${LOCAL_TENANT_PORT}
    echo export JC_TOGGLE_API_URL=http://localhost:${LOCAL_TOGGLE_PORT}/api
    echo export JC_AUTH_URL=https://auth.prod-preview.openshift.io
    echo export JC_SERVICE_ACCOUNT_ID=${JC_SERVICE_ACCOUNT_ID}
    echo export JC_SERVICE_ACCOUNT_SECRET=\"${JC_SERVICE_ACCOUNT_SECRET}\"
    echo export JC_AUTH_TOKEN_KEY=\"${JC_AUTH_TOKEN_KEY}\"
    echo export JC_FIXED_UUIDS=${JC_FIXED_UUIDS}
}

###############################################################################
# Unsets the exported Idler environment variables.
# Globals:
#   None
# Arguments:
#   None
# Returns:
#   None
###############################################################################
unsetEnv() {
    echo unset JC_IDLE_AFTER
    echo unset JC_JENKINS_PROXY_API_URL
    echo unset JC_F8TENANT_API_URL
    echo unset JC_TOGGLE_API_URL
    echo unset JC_AUTH_URL
    echo unset JC_SERVICE_ACCOUNT_ID
    echo unset JC_SERVICE_ACCOUNT_SECRET
    echo unset JC_AUTH_TOKEN_KEY
}

###############################################################################
# Stops oc-port forwarding.
# Globals:
#   None
# Arguments:
#   None
# Returns:
#   None
###############################################################################
stop() {
    pids=$(pgrep -a -f -d " " "setupLocalIdler.sh start")
    pids+=$(pgrep -a -f -d " " "oc --config $(dirname $0)/config")
    kill -9 ${pids}
}

case "$1" in
  start)
    start
    ;;
  stop)
    stop
    ;;
  env)
    env
    ;;
  unset)
    unsetEnv
    ;;
  *)
    printHelp
esac
