versionOfKustomizeCLI := v3.2.3

install:
	./scripts/buildKustomizeCLI.sh ${versionOfKustomizeCLI}
	./scripts/build.sh
