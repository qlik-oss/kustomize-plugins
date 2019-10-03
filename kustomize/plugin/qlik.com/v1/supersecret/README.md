# SuperSecret Kustomize Plugin

## Using as a transformer:

When used as a `transformer`, the plugin will optionally append a name suffix hash for the specified secret unless specifically disabled by setting `disableNameSuffixHash: true`.
It will also optionally append data to the secret from the contents of the `stringData` map. Name suffix hash is calculated after any updates to the secret's data.   

Create a layout that looks like this:
```text
tree .
.
├── deployment.yaml
├── kustomization.yaml
├── secret.yaml
└── secretTransformer.yaml
```

```bash
cat <<'EOF' >kustomization.yaml
resources:
- deployment.yaml
- secret.yaml

transformers:
- secretTransformer.yaml
EOF
```

```bash
cat <<'EOF' >deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: my-pod
        image: some-image
        volumeMounts:
        - name: foo
          mountPath: "/etc/foo"
          readOnly: true
      volumes:
      - name: foo
        secret:
          secretName: my-secret
EOF
```

```bash
cat <<'EOF' >secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-secret
type: Opaque
data:
  initial: d2hhdGV2ZXI=
EOF
```

```bash
cat <<'EOF' >secretTransformer.yaml
apiVersion: qlik.com/v1
kind: SuperSecret
metadata:
  name: my-secret
stringData:
  foo: bar
  baz: whatever
EOF
```

Get and build the plugins:
```bash
git clone git@github.com:qlik-oss/kustomize-plugins.git
pushd kustomize-plugins
git checkout SecretHashTransformer
make install
popd
```

Then run `kustomize` on the directory:
```bash
XDG_CONFIG_HOME=kustomize-plugins $HOME/go/bin/kustomize build --enable_alpha_plugins .
```

The output will look like so:
```text
apiVersion: v1
data:
  baz: d2hhdGV2ZXI=
  foo: YmFy
  initial: d2hhdGV2ZXI=
kind: Secret
metadata:
  name: my-secret-b22th6mh4g
type: Opaque
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment
spec:
  replicas: 3
  template:
    spec:
      containers:
      - image: some-image
        name: my-pod
        volumeMounts:
        - mountPath: /etc/foo
          name: foo
          readOnly: true
      volumes:
      - name: foo
        secret:
          secretName: my-secret-b22th6mh4g
```

## Using as a generator:

When used as a `generator`, the plugin will generate a secret with the specified name.
It will also optionally append a name suffix hash unless specifically disabled by setting `disableNameSuffixHash: true`.
The data of the secret will be set from the contents of the `stringData` map.

Create a layout that looks like this:
```text
tree .
.
├── deployment.yaml
├── kustomization.yaml
├── secretGenerator.yaml
```

```bash
cat <<'EOF' >kustomization.yaml
resources:
- deployment.yaml

generators:
- secretGenerator.yaml
EOF
```

```bash
cat <<'EOF' >deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: my-pod
        image: some-image
        volumeMounts:
        - name: foo
          mountPath: "/etc/foo"
          readOnly: true
      volumes:
      - name: foo
        secret:
          secretName: my-secret
EOF
```

```bash
cat <<'EOF' >secretGenerator.yaml
apiVersion: qlik.com/v1
kind: SuperSecret
metadata:
  name: my-secret
stringData:
  foo: bar
  baz: whatever
EOF
```

Get and build the plugins:
```bash
git clone git@github.com:qlik-oss/kustomize-plugins.git
pushd kustomize-plugins
git checkout SecretHashTransformer
make install
popd
```

Then run `kustomize` on the directory:
```bash
XDG_CONFIG_HOME=kustomize-plugins $HOME/go/bin/kustomize build --enable_alpha_plugins .
```

The output will look like so:
```text
apiVersion: v1
data:
  baz: d2hhdGV2ZXI=
  foo: YmFy
kind: Secret
metadata:
  name: my-secret-k8gb8gd84f
type: Opaque
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment
spec:
  replicas: 3
  template:
    spec:
      containers:
      - image: some-image
        name: my-pod
        volumeMounts:
        - mountPath: /etc/foo
          name: foo
          readOnly: true
      volumes:
      - name: foo
        secret:
          secretName: my-secret-k8gb8gd84f
```