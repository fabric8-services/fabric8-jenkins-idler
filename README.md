# Jenkins Controller

This service watches builds in OpenShift and idles/unidles Jenkins for given namespaces when needed

It requires following secrets and configmaps living in different repos to be applied:

* https://github.com/fabric8-services/fabric8-tenant/blob/master/f8tenant.secrets.yaml (uses `openshift.tenant.masterurl`)
* ... (#FIXME)

## Problem Statement

There is a Jenkins instance running for each tenant in OpenShift.io. Currently, these instance are up all the time which means `# of tenant * 1 GB of memory` allocated even when there is nothing happening in any of Jenkins instance.

Goal for this (fabric8-jenkins-idler) service is to watch for activity and idle/unidle Jenkins for tenants as needed.

There are couple triggeres which might need Jenkins to be unidled:

* (Github) Webhooks
* User accessing Jenkins UI/API
* A build started in OpenShift

### (Github) Webhooks

**Problem:** Github sets 10s timeout for webhooks, which means that if Jenkins does not respond with `200` in 10s, webhook call will be considered failed
**Proposed Solution:** Create a webhook proxy which will accept the request, verify user & jenkins exist and return `200` (if it does exist) or `404` (if it does not). Then it will buffer the request and un-idle Jenkins. Once Jenkins is running, it'll re-fire the request to the Jenkins to kick off the build.
**Another Solution:** Don't use Jenkins webhooks - OpenShift provides it's own build webhooks and thus we could configure those instead of Jenkins ones. Problem with this is that Fabric8 would be locked into OpenShift and could not run on plain Kubernetes (or other orchastrations which do not support these build webhooks)

### User accessing Jenkins UI/API

**Problem:** There are actually 2 problems here:

1. Jenkins could get idled while user is using it.
2. User experience is not good when unidling slowly-starting service (Jenkins takes 1-1.5 minutes to start)

First issue might occure when user is looking into UI while a build is finished for some time already and idling event from idler happens. As we have (currently) no way to know a user is accessing the UI/API, we'd idle based on finished builds and the UI/API would went down under user's hands.

Second issue is that user gets `50x` error from OpenShift when a service is idled (or does not exist at all). That is fine for webservices which can start in up to seconds - timeout is long enough to keep the connection up and wait for response. But as Jenkins often takes over a minute to start, the connection times out and it looks like Jenkins does not exist at all to the user.

**Proposed Solution:** Create a proxy (similar to webhooks) which will return a loading page stating Jenkins is starting and regularly poll the proxy API to see if Jenkins is ready. Once it's up, it can redirect (not nice UX) or proxy (needs auth from user and token swap in headers - OSIO token -> OSO token) the Jenkins content.

### A build starter in OpenShift

**Problem:** OpenShift has a notion of builds (with its BuildConfig and Build objects). These objects are followed by a Jenkins plugin which then, based on changes in OpenShift objects, performs actions in Jenkins. The problem for us is the nature of implementation - Jenkins is polling OpenShift for information and pushing information back. So when you start a build in OpenShift while Jenkins is idled (i.e. is not running) nothing will happen and the build will hang there indefinitely.

**Proposed Soltion:** Let's create a different service which will follow builds for all users and unidle Jenkins if there is a new build coming. At the same time, it will idle Jenkins when there is not build/activity happening for a long time.