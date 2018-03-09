# Problem Statement

<!-- MarkdownTOC -->

- [Webhooks](#webhooks)
	- [Problem](#problem)
	- [Proposed solution](#proposed-solution)
- [User accessing the Jenkins UI/API](#user-accessing-the-jenkins-uapi)
	- [Problem](#problem-1)
	- [Proposed solution](#proposed-solution-1)
- [A build starter in OpenShift](#a-build-starter-in-openshift)
	- [Problem](#problem-2)
	- [Proposed solution](#proposed-solution-2)

<!-- /MarkdownTOC -->

Each tenant of OpenShift.io has a Jenkins instance.
These instances when running consume 1 GB per tenant, even if there is no activity.

The Jenkins Idler service watches for activity and idles respectively unidles a tenant's Jenkins instances as needed.

There are several conditions which can trigger an unidle event:

* Github webhooks
* User accessing the Jenkins UI/API
* A Build started in OpenShift

<a name="webhooks"></a>
## Webhooks

<a name="problem"></a>
### Problem

GitHub's timeout for webhook delivery is 10 seconds.
If GitHub does not receive a HTTP status 200 within this time frame, the webhook delivery will be marked as failed.

<a name="proposed-solution"></a>
### Proposed solution

Create a webhook proxy which will accept the request, verify tenant & Jenkins instance exist and return `200` (if it does exist) or `404` (if it does not).
The proxy will buffer the request and unidle Jenkins.
Once Jenkins is running, the original request is replayed to Jenkins to start the build.

<a name="user-accessing-the-jenkins-uapi"></a>
## User accessing the Jenkins UI/API

<a name="problem-1"></a>
### Problem

1. Jenkins could get idled while user is using it.
1. User experience is not good when unidling, because Jenkins takes several minutes to start.

The first issue might occur when user is looking into UI while a build is finished for some time already and idling event from idler happens.
As we have (currently) no way to know a user is accessing the UI/API, we'd idle based on finished builds and the UI/API would went down under user's hands.

Second issue is that user gets `50x` error from OpenShift when a service is idled (or does not exist at all).
That is fine for web-services which can start in up to seconds - timeout is long enough to keep the connection up and wait for response.
But as Jenkins often takes minutes to start, the connection times out and it looks like Jenkins does not exist at all to the user.

<a name="proposed-solution-1"></a>
### Proposed solution

Create a proxy (similar to webhooks) which will return a loading page stating Jenkins is starting.
It also polls Jenkins to determine when the startup is complete.
Once Jenkins is up, the proxy can redirect to Jenkins.

<a name="a-build-starter-in-openshift"></a>
## A build starter in OpenShift

<a name="problem-2"></a>
### Problem

OpenShift has a notion of Builds (with its `BuildConfig` and `Build` objects).
These objects are followed by a Jenkins plugin which then, based on changes in OpenShift objects, performs actions in Jenkins.
The problem for us is the nature of its implementation - Jenkins is polling OpenShift for information and pushing information back.
So when you start a build in OpenShift while Jenkins is idled nothing will happen and the build will hang indefinitely.

<a name="proposed-solution-2"></a>
### Proposed solution

Let's create a different service which will follow builds for all users and unidle Jenkins if there is a new Build.
At the same time, it will idle Jenkins when there is not build/activity happening for a long time.
