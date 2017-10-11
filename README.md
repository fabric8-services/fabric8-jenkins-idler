# Jenkins Controller

This service watches builds in OpenShift and idles/unidles Jenkins for given namespaces when needed

It requires following secrets and configmaps living in different repos to be applied:

* https://github.com/fabric8-services/fabric8-tenant/blob/master/f8tenant.secrets.yaml (uses `openshift.tenant.masterurl`)
* ... (#FIXME)
