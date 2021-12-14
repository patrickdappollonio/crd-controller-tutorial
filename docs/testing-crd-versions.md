# Testing Kubernetes CRD versions

One interesting thing to see when working with CRDs `storage` and `served` flags is that you can actually query different versions of the same resource. 

This example will work in Kubernetes above 1.15 and below 1.21. If you're using `kind`, you should get a cluster, at the time of writing this document, with Kubernetes 1.21.

Copy the ["Minimal example" from the Ingress documentation page of Kubernetes](https://kubernetes.io/docs/concepts/services-networking/ingress/#the-ingress-resource), copied below:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: minimal-ingress
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  rules:
  - http:
      paths:
      - path: /testpath
        pathType: Prefix
        backend:
          service:
            name: test
            port:
              number: 80
```

Now we know it won't work because we don't have a `Service` called `test`, but we will use it to show some interesting behaviour.

On Kubernetes 1.21, when you query an `Ingress`, you might get the following message:

```
Warning: extensions/v1beta1 Ingress is deprecated in v1.14+, unavailable in v1.22+; use networking.k8s.io/v1 Ingress
```

We can use it to play with it and versions of the ingress above. Save the contents of the ingress into a file called `ingress.yaml`, then apply it:

```
$ kubectl apply -f ingress.yaml
ingress.networking.k8s.io/minimal-ingress created
```

Now, let's query multiple versions of this object. Since the original one has the following version data:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
```

We can use that, plus `kubectl`, to query it especifically by version. The `kubectl get` help content says that the usage can be either of the following (see `kubectl get -h`):

```bash
kubectl get TYPE[.VERSION][.GROUP] [NAME | -l label] 
kubectl get TYPE[.VERSION][.GROUP]/NAME
```

So let's use the first one, which is:

```bash
$ kubectl get ingress.v1.networking.k8s.io minimal-ingress
NAME              CLASS    HOSTS   ADDRESS   PORTS   AGE
minimal-ingress   <none>   *                 80      11m
```

This works. How about...

```bash
$ kubectl get ingress.v1beta1.extensions minimal-ingress
Warning: extensions/v1beta1 Ingress is deprecated in v1.14+, unavailable in v1.22+; use networking.k8s.io/v1 Ingress
NAME              CLASS    HOSTS   ADDRESS   PORTS   AGE
minimal-ingress   <none>   *                 80      11m
```

... This also works! Although we received a warning, we are still able to query the version of the Ingress using an old CRD version of it. Let's go one step further. Let's request the resources as YAML (and we will use a quick `grep` to only show the `spec` part, excluding the `metadata`). For the first one we have:


```bash
kubectl get ingress.v1.networking.k8s.io minimal-ingress -o yaml | grep -A 100 "spec:"
```
```yaml
spec:
  rules:
  - http:
      paths:
      - backend:
          service:
            name: test
            port:
              number: 80
        path: /testpath
        pathType: Prefix
status:
  loadBalancer: {}
```

As expected, we get the response we were hoping for. I mean, we created an Ingress using this specific version, version `v1` of the group `networking.k8s.io`. But since we were able to retrieve it before using the old version, will we be able to retrieve the YAML for this old version too? Will it be like old ingresses or will it be like the new ingresses? Let's see:

```bash
kubectl get ingress.v1beta1.extensions minimal-ingress -o yaml | grep -A 100 "spec:"
```

```yaml
Warning: extensions/v1beta1 Ingress is deprecated in v1.14+, unavailable in v1.22+; use networking.k8s.io/v1 Ingress
spec:
  rules:
  - http:
      paths:
      - backend:
          serviceName: test
          servicePort: 80
        path: /testpath
        pathType: Prefix
status:
  loadBalancer: {}
```

Wait... This doesn't look at all how we submitted it ourselves! We defined a `spec.rules[0].http.paths[0].backend.service.name` with the value of `test`, not a `spec.rules[0].http.paths[0].backend.serviceName`!

Was this done by the API or by `kubectl`? Let's see. Let's proxy the API locally:

```bash
kubectl proxy
```

And now, let's run the previous commandds with verbose mode enabled at level 8, and grab that log output sent to `stderr`, pipe it to `stdout` and find the `HTTP GET` call done by `kubectl`:

```bash
kubectl get ingress.v1.networking.k8s.io minimal-ingress -v=8 2>&1 | grep "minimal-ingress"
```

```
I1214 16:19:47.039228   75843 round_trippers.go:432] GET https://127.0.0.1:37691/apis/networking.k8s.io/v1/namespaces/default/ingresses/minimal-ingress
minimal-ingress   <none>   *                 80      23m
```

And now let's call that endpoint, but against our proxy:

```bash
curl -s localhost:8001/apis/networking.k8s.io/v1/namespaces/default/ingresses/minimal-ingress | jq .spec
```

```json
{
  "rules": [
    {
      "http": {
        "paths": [
          {
            "path": "/testpath",
            "pathType": "Prefix",
            "backend": {
              "service": {
                "name": "test",
                "port": {
                  "number": 80
                }
              }
            }
          }
        ]
      }
    }
  ]
}
```

Ok, looks good. Let's look at the new one, which we can guess the URL, but let's do it still nonetheless. First, get the API Endpoint call using `kubectl` verbose mode:

```bash
kubectl get ingress.v1beta1.extensions minimal-ingress -v=8 2>&1 | grep "minimal-ingress"
```

```
I1214 16:21:07.989565   76368 round_trippers.go:432] GET https://127.0.0.1:37691/apis/extensions/v1beta1/namespaces/default/ingresses/minimal-ingress
minimal-ingress   <none>   *                 80      24m
```

Then let's replace the host for our proxy, and make the call:

```bash
curl -s localhost:8001/apis/extensions/v1beta1/namespaces/default/ingresses/minimal-ingress | jq .spec
```

```json
{
  "rules": [
    {
      "http": {
        "paths": [
          {
            "path": "/testpath",
            "pathType": "Prefix",
            "backend": {
              "serviceName": "test",
              "servicePort": 80
            }
          }
        ]
      }
    }
  ]
}
```

So `kubectl` didn't do the "conversion", the Kubernetes API did!

Keep in mind this is one edge case scenario [of which not many people are happy](https://github.com/kubernetes/kubernetes/issues/94761). The standard is that custom resources should create their own Conversion logic [using a Conversion Webhook](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#webhook-conversion). No conversion is done automagically for 3rd party resources.

Ingresses are handled differently, just because they are ingresses. In better words:

> When you create an ingress object, it can be read via any version (the server handles converting into the requested version). `kubectl get ingress` is an ambiguous request, since it does not indicate what version is desired to be read.
> 
> When an ambiguous request is made, kubectl searches the discovery docs returned by the server to find the first group/version that contains the specified resource.
> 
> For compatibility reasons, extensions/v1beta1 has historically been preferred over all other api versions. Now that ingress is the only resource remaining in that group, and is deprecated and has a GA replacement, 1.20 will drop it in priority so that `kubectl get ingress` would read from `networking.k8s.io/v1`, but a 1.19 server will still follow the historical priority.
> 
> If you want to read a specific version, you can qualify the get request (like `kubectl get ingresses.v1.networking.k8s.io ...`) or can pass in a manifest file to request the same version specified in the file (`kubectl get -f ing.yaml -o yaml`)
> [#](https://github.com/kubernetes/kubernetes/issues/94761#issuecomment-691982480).

It's also worth nothing that:

> The API version used to create an object is not intentionally exposed in the API... all ingress objects are available via all served apiVersions, regardless of the version they were created with (the API server converts between versions in order to return ingress objects in the requested version).
> [#](https://github.com/kubernetes/kubernetes/issues/94761#issuecomment-880900951)

On all other third-party cases, the [standard Kubernetes CRD versioning rules](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#overview) remain in place.

For `storage:`, when a version is marked as such, it's used like that to be serialized into `etcd`, then:

> Whenever you fetch an object, the api server reads it from etcd, converts it into an internal version, then converts it to the version you requested. [#](https://github.com/kubernetes/kubernetes/issues/58131#issuecomment-404466779)

It's also worth noting that:

> Controllers always get the version they requested from the API. They are not exposed to the stored version. [#](https://github.com/kubernetes/kubernetes/issues/58131#issuecomment-404466779)

There's an entire [Google Docs Document](https://docs.google.com/document/d/1eoS1K40HLMl4zUyw5pnC05dEF3mzFLp5TPEEt4PFvsM/edit) regarding this discussion and how CRDs and "Stored" vs "Served" works.
