# SedOnPath Kustomize Plugin

If what's at the end of the specified path is a:
1) `string`, then the transformer will execute the provided sed-expression on it;
2) `[]string`, then the transformer will execute the provided sed-expression on `each string` in the array;
3) `anything else`, throw error

NOTE: The plugin shells out to `sed`. On Mac, you'll need `gnu-sed`. To install it, do the following:
```bash
brew install gnu-sed
ln -s /usr/local/bin/gsed /usr/local/bin/sed
```