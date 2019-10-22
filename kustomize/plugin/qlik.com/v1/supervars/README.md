# SuperVars Kustomize Plugin

## Example usage:

Create a layout that looks like this:
```text
tree .
.
├── kustomization.yaml
└── resources
    ├── configmap.yaml
    ├── kustomization.yaml
    ├── secret.yaml
    ├── superVars.yaml
    └── varreference.yaml
```

```bash
cat <<'EOF' >kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
generators:
- resources
EOF
```


```bash
mkdir resources
```

```bash
cat <<'EOF' >resources/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- secret.yaml
- configmap.yaml
transformers:
- superVars.yaml
EOF
```

```bash
cat <<'EOF' >resources/secret.yaml
apiVersion: qlik.com/v1
kind: SuperSecret
metadata:
  name: my-secret
  labels:
    myproperty: propertyvalue
stringData:
  myproperty: $(MYPROPERTY)-something
EOF
```

```bash
cat <<'EOF' >resources/configmap.yaml
apiVersion: qlik.com/v1
kind: SuperConfigMap 
metadata:
  name: my-configmap
  labels:
    myproperty: propertyvalue-2
data:
  myproperty: $(MYPROPERTY2)-something
EOF
```

```bash
cat <<'EOF' >resources/superVars.yaml
apiVersion: qlik.com/v1
kind: SuperVars 
metadata:
  name: notImportantHere
configurations:
- varreference.yaml
vars:
- name: MYPROPERTY
  objref:
    apiVersion: qlik.com/v1
    kind: SuperSecret
    name: my-secret
  fieldref:
    fieldpath: metadata.labels.myproperty 
- name: MYPROPERTY2
  objref:
    apiVersion: qlik.com/v1
    kind: SuperConfigMap 
    name: my-configmap
  fieldref:
    fieldpath: metadata.labels.myproperty 
EOF
```

```bash
cat <<'EOF' >resources/varreference.yaml
varReference:
- path: stringData/myproperty
  kind: SuperSecret 
- path: data/myproperty
  kind: SuperConfigMap 
EOF
```

Get and build the plugins:
```bash
git clone git@github.com:qlik-oss/kustomize-plugins.git
pushd kustomize-plugins
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
  myproperty: propertyvalue-2-something
kind: ConfigMap
metadata:
  name: my-configmap-fk425ft945
---
apiVersion: v1
data:
  myproperty: cHJvcGVydHl2YWx1ZS1zb21ldGhpbmc=
kind: Secret
metadata:
  name: my-secret-27g4c6m9fh
type: Opaque
```

And variable inside the secret should be resolved as well: 
```bash
printf "cHJvcGVydHl2YWx1ZS1zb21ldGhpbmc=" | base64 -D
propertyvalue-something
```