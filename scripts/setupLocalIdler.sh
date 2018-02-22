#!/usr/bin/env bash
#
# Used to run Idler locally.
#

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
Usage: ${0##*/} [start|stop]

This script is used to run the Jenkins Idler on localhost.
As a prerequisite DSAAS_PREVIEW_TOKEN, JC_OPENSHIFT_API_TOKEN and  JC_AUTH_TOKEN need to be exported
in the shell you run this script.

DSAAS_PREVIEW_TOKEN is the OpenShift token for console.rh-idev.openshift.com, JC_OPENSHIFT_API_TOKEN is
the service account token used by the Idler and  JC_AUTH_TOKEN is the authentication token for the auth service
used by the Idler.

Sample usage (in your shell from the root of fabric8-jenkins-idler):

> export DSAAS_PREVIEW_TOKEN=<dsaas-preview token>
> export JC_OPENSHIFT_API_TOKEN=<OpenShift API token>
> export JC_AUTH_TOKEN=<auth token>
> export JC_FIXED_UUIDS=<list of uuids to enable for idling>
> ./scripts/${0##*/} start
> eval \$(./scripts/${0##*/} env)
> fabric8-jenkins-idler

To stop:

> ./scripts/$(${0##*/}) start
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
    oc -n dsaas-preview --token ${DSAAS_PREVIEW_TOKEN} $@
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
# Starts the port forwarding.
# Globals:
#   None
# Arguments:
#   None
# Returns:
#   None
###############################################################################
start() {
    [ -z "${DSAAS_PREVIEW_TOKEN}" ] && printHelp && exit 1
    [ -z "${JC_OPENSHIFT_API_TOKEN}" ] && printHelp && exit 1
    [ -z "${JC_AUTH_TOKEN}" ] && printHelp && exit 1

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
    [ -z "${JC_OPENSHIFT_API_TOKEN}" ] && printHelp && exit 1
    [ -z "${JC_AUTH_TOKEN}" ] && printHelp && exit 1
    [ -z "${JC_FIXED_UUIDS}" ] && printHelp && exit 1

    echo export JC_OPENSHIFT_API_URL=https://api.free-stg.openshift.com
    echo export JC_OPENSHIFT_API_TOKEN=${JC_OPENSHIFT_API_TOKEN}
    echo export JC_AUTH_TOKEN=${JC_AUTH_TOKEN}
    echo export JC_IDLE_AFTER=${LOCAL_DEFAULT_IDLE_TIME}
    echo export JC_JENKINS_PROXY_API_URL=http://localhost:${LOCAL_PROXY_PORT}
    echo export JC_F8TENANT_API_URL=http://localhost:${LOCAL_TENANT_PORT}
    echo export JC_TOGGLE_API_URL=http://localhost:${LOCAL_TOGGLE_PORT}/api
    echo export JC_FIXED_UUIDS=${JC_FIXED_UUIDS}
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
    pids+=$(pgrep -a -f -d " " "oc -n dsaas-preview --token")
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
  *)
    printHelp
esac

