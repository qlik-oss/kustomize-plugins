versionOfKustomizeCLI := v3.4.0

install:
	./scripts/buildKustomizeCLI.sh ${versionOfKustomizeCLI}
	./scripts/build.sh
