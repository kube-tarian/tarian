# Troubleshooting

## Sidecar injection problems

Tarian pod agent runs as a sidecar container in the monitored pod, watching and validating processes and files based on the registered constraints. In some cases the sidecar is not successfully injected. This describes how to resolve it.

### Understanding how a sidecar is injected

Firstly, it is really helpful to understand how the pod-agent sidecar is injected. Kubernetes has [admission controllers](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/) which intercept requests to the Kubernetes API server prior to the persistence of the object. The mutating admission controllers may validate and modify the objects they admit. 

Kubernetes provides a way to extend the admission controller via webhook through [MutatingAdmissionWebhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/). The webhook is registered using the [MutatingWebhookConfiguration](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#webhook-configuration) API object. In tarian, it's installed as a part of tarian-cluster-agent.

```bash
kubectl get MutatingWebhookConfiguration

kubectl get MutatingWebhookConfiguration tarian-cluster-agent -o yaml
```

If you use the default installation, the contents of the webhook configuration would be:

```yaml
webhooks:
- admissionReviewVersions:
  - v1
  clientConfig:
    service:
      name: tarian-cluster-agent-controller-manager
      namespace: tarian-system
      path: /inject-pod-agent
      port: 443
  rules:
  - apiGroups:
    - ""
    apiVersions:
    - v1
    operations:
    - CREATE
    resources:
    - pods
    scope: '*'
```

This means, on every pod created the webhook controller will send a webhook request to `tarian-cluster-agent-controller-manager.tarian-system.svc`, which is served by pods from `tarian-cluster-agent-controller-manager` deployment. These pods will modify the pod's spec to include a sidecar container (pod-agent).

### Sidecar is not injected silently (no error)

Enable log-level=debug to see more verbose logs.

```bash
helm upgrade tarian-cluster-agent tarian/tarian-cluster-agent --devel -n tarian-system --set clusterAgent.log.level=debug
```

After the rollout is completed, you should see more logs in tarian-cluster-agent-controller-manager:

```bash
kubectl logs deploy/tarian-cluster-agent-controller-manager -n tarian-system -f
```

Now when a webhook request comes, a log `handling a webhook request` should show up there. If there is no such log, a webhook is not coming to the pod.
See the next sections for what could cause a webhook not coming to the service.


### The admission controller cannot connect to the webhook service

The default failurePolicy in tarian-cluster-agent MutatingWebhookConfiguration is `Ignore`, which means it ignores the failure and the pod will still be created without mutation (sidecar is not injected silently). While this is good to prevent issue in production, it also hides the error message that's useful for debugging.

To debug, you can temporarily edit the tarian-cluster-agent MutatingWebhookConfiguration and set the failurePolicy to `Fail` and limit the scope to a specific
namespace so that other namespaces can still work.


```bash
kubectl edit MutatingWebhookConfiguration tarian-cluster-agent
```

change to this:

```yaml
webhooks:
- failurePolicy: Fail
  namespaceSelector:
    matchExpressions:
    - key: kubernetes.io/metadata.name
      operator: In
      values: ["default"]
  rules:
  - apiGroups:
    - ""
    apiVersions:
    - v1
    operations:
    - CREATE
    resources:
    - pods
    scope: "Namespaced"
```

Now when a pod is created and the webhook fails, an error will be returned. For example:

```
Error from server (InternalError): error when creating "dev/config/monitored-pod/register.yaml": Internal error occurred: failed calling webhook "tarian-cluster-agent.k8s.tarian.dev": Post "https://tarian-cluster-agent-controller-manager.tarian-system.svc:443/inject-pod-agent?timeout=10s": dial tcp 10.128.245.250:443: connect: connection refused
```

Then we check why the connection was refused. Was the pod ready? Was there a network policy issue? Were there any firewall issues?


### Certificate related errors


```
Error from server (InternalError): error when creating "dev/config/monitored-pod/register.yaml": Internal error occurred: failed calling webhook "tarian-cluster-agent.k8s.tarian.dev": Post "https://tarian-cluster-agent-controller-manager.tarian-system.svc:443/inject-pod-agent?timeout=10s": x509: certificate signed by unknown authority
```

Tarian webhook server (deployment name: `tarian-cluster-agent-controller-manager`) by default manages the certificate needed by the webhook admission controller. It's configured in the tarian-cluster-agent `MutatingWebhookConfiguration`, in `caBundle`. In some rare conditions, the `caBundle` field might not be updated yet so that the CA doesn't match with the one used by the webhook server.

If that's the case, you can try to delete the pod in tarian-cluster-agent-controller-manager deployment. A new pod will be created and will try update the caBundle.